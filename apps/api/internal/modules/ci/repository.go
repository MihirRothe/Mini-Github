package ci

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/google/uuid"
)

type RepositoryStore interface {
	CreatePipelineRun(ctx context.Context, run *PipelineRun) error
	GetPipelineRun(ctx context.Context, runID string) (*PipelineRun, error)
	ListPipelineRuns(ctx context.Context, repoID string, filter RunFilter) ([]*PipelineRun, int, error)
	UpdatePipelineRunStatus(ctx context.Context, runID string, status PipelineStatus, startedAt, finishedAt *time.Time) error

	CreatePipelineJob(ctx context.Context, job *PipelineJob) error
	GetPipelineJob(ctx context.Context, jobID string) (*PipelineJob, error)
	ListJobsForRun(ctx context.Context, runID string) ([]*PipelineJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, exitCode *int, startedAt, finishedAt *time.Time) error

	CreateJobStep(ctx context.Context, step *JobStep) error
	UpdateStepStatus(ctx context.Context, stepID string, status StepStatus, exitCode *int, startedAt, finishedAt *time.Time) error

	AppendJobLog(ctx context.Context, log *JobLog) error
	GetJobLogs(ctx context.Context, jobID string, afterLine int) ([]*JobLog, error)
}

func NewRepositoryStore(db *database.DB, authRepo auth.Repository) RepositoryStore {
	if db == nil || db.IsStandalone() {
		return NewMemoryRepositoryStore(authRepo)
	}
	return &sqlRepositoryStore{db: db, authRepo: authRepo}
}

// -------------------------------------------------------------------------
// SQL Implementation (PostgreSQL)
// -------------------------------------------------------------------------

type sqlRepositoryStore struct {
	db       *database.DB
	authRepo auth.Repository
}

func (s *sqlRepositoryStore) CreatePipelineRun(ctx context.Context, run *PipelineRun) error {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	run.CreatedAt = now
	run.UpdatedAt = now

	query := `
		INSERT INTO pipeline_runs (
			id, repository_id, workflow_name, workflow_file, trigger_event,
			trigger_user_id, commit_sha, commit_message, branch, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.RepositoryID, run.WorkflowName, run.WorkflowFile, run.TriggerEvent,
		run.TriggerUserID, run.CommitSHA, run.CommitMessage, run.Branch, string(run.Status),
		run.CreatedAt, run.UpdatedAt,
	)
	return err
}

func (s *sqlRepositoryStore) GetPipelineRun(ctx context.Context, runID string) (*PipelineRun, error) {
	query := `
		SELECT id, repository_id, workflow_name, workflow_file, trigger_event,
		       trigger_user_id, commit_sha, commit_message, branch, status,
		       started_at, finished_at, created_at, updated_at
		FROM pipeline_runs
		WHERE id = $1
	`
	var run PipelineRun
	var statusStr string
	var triggerUserID sql.NullString
	var commitMsg sql.NullString

	err := s.db.QueryRowContext(ctx, query, runID).Scan(
		&run.ID, &run.RepositoryID, &run.WorkflowName, &run.WorkflowFile, &run.TriggerEvent,
		&triggerUserID, &run.CommitSHA, &commitMsg, &run.Branch, &statusStr,
		&run.StartedAt, &run.FinishedAt, &run.CreatedAt, &run.UpdatedAt,
	)
	if err != nil {
		return nil, ErrRunNotFound
	}

	run.Status = PipelineStatus(statusStr)
	if triggerUserID.Valid {
		run.TriggerUserID = &triggerUserID.String
		if u, _ := s.authRepo.GetUserByID(ctx, *run.TriggerUserID); u != nil {
			pub := u.ToPublic()
			run.TriggerUser = &pub
		}
	}
	if commitMsg.Valid {
		run.CommitMessage = commitMsg.String
	}

	if run.StartedAt != nil && run.FinishedAt != nil {
		run.DurationMs = run.FinishedAt.Sub(*run.StartedAt).Milliseconds()
	}

	return &run, nil
}

func (s *sqlRepositoryStore) ListPipelineRuns(ctx context.Context, repoID string, filter RunFilter) ([]*PipelineRun, int, error) {
	baseQuery := `
		FROM pipeline_runs
		WHERE repository_id = $1
	`
	args := []any{repoID}
	argIdx := 2

	if filter.Status != "" && filter.Status != "all" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Branch != "" {
		baseQuery += fmt.Sprintf(" AND branch = $%d", argIdx)
		args = append(args, filter.Branch)
		argIdx++
	}
	if filter.Event != "" {
		baseQuery += fmt.Sprintf(" AND trigger_event = $%d", argIdx)
		args = append(args, filter.Event)
		argIdx++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	selectQuery := `
		SELECT id, repository_id, workflow_name, workflow_file, trigger_event,
		       trigger_user_id, commit_sha, commit_message, branch, status,
		       started_at, finished_at, created_at, updated_at
	` + baseQuery + fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", pageSize, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []*PipelineRun
	for rows.Next() {
		var run PipelineRun
		var statusStr string
		var triggerUserID sql.NullString
		var commitMsg sql.NullString

		if err := rows.Scan(
			&run.ID, &run.RepositoryID, &run.WorkflowName, &run.WorkflowFile, &run.TriggerEvent,
			&triggerUserID, &run.CommitSHA, &commitMsg, &run.Branch, &statusStr,
			&run.StartedAt, &run.FinishedAt, &run.CreatedAt, &run.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		run.Status = PipelineStatus(statusStr)
		if triggerUserID.Valid {
			run.TriggerUserID = &triggerUserID.String
			if u, _ := s.authRepo.GetUserByID(ctx, *run.TriggerUserID); u != nil {
				pub := u.ToPublic()
				run.TriggerUser = &pub
			}
		}
		if commitMsg.Valid {
			run.CommitMessage = commitMsg.String
		}
		if run.StartedAt != nil && run.FinishedAt != nil {
			run.DurationMs = run.FinishedAt.Sub(*run.StartedAt).Milliseconds()
		}

		runs = append(runs, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

func (s *sqlRepositoryStore) UpdatePipelineRunStatus(ctx context.Context, runID string, status PipelineStatus, startedAt, finishedAt *time.Time) error {
	query := `
		UPDATE pipeline_runs
		SET status = $1, started_at = COALESCE($2, started_at), finished_at = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := s.db.ExecContext(ctx, query, string(status), startedAt, finishedAt, runID)
	return err
}

func (s *sqlRepositoryStore) CreatePipelineJob(ctx context.Context, job *PipelineJob) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	query := `
		INSERT INTO pipeline_jobs (
			id, pipeline_run_id, job_key, name, runs_on, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.PipelineRunID, job.JobKey, job.Name, job.RunsOn, string(job.Status),
		job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (s *sqlRepositoryStore) GetPipelineJob(ctx context.Context, jobID string) (*PipelineJob, error) {
	query := `
		SELECT id, pipeline_run_id, job_key, name, runs_on, status,
		       started_at, finished_at, exit_code, created_at, updated_at
		FROM pipeline_jobs
		WHERE id = $1
	`
	var job PipelineJob
	var statusStr string
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.PipelineRunID, &job.JobKey, &job.Name, &job.RunsOn, &statusStr,
		&job.StartedAt, &job.FinishedAt, &job.ExitCode, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, ErrJobNotFound
	}
	job.Status = JobStatus(statusStr)
	if job.StartedAt != nil && job.FinishedAt != nil {
		job.DurationMs = job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
	}

	steps, err := s.listStepsForJob(ctx, job.ID)
	if err == nil {
		job.Steps = steps
	}
	return &job, nil
}

func (s *sqlRepositoryStore) ListJobsForRun(ctx context.Context, runID string) ([]*PipelineJob, error) {
	query := `
		SELECT id, pipeline_run_id, job_key, name, runs_on, status,
		       started_at, finished_at, exit_code, created_at, updated_at
		FROM pipeline_jobs
		WHERE pipeline_run_id = $1
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*PipelineJob
	for rows.Next() {
		var job PipelineJob
		var statusStr string
		if err := rows.Scan(
			&job.ID, &job.PipelineRunID, &job.JobKey, &job.Name, &job.RunsOn, &statusStr,
			&job.StartedAt, &job.FinishedAt, &job.ExitCode, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		job.Status = JobStatus(statusStr)
		if job.StartedAt != nil && job.FinishedAt != nil {
			job.DurationMs = job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
		}
		jobs = append(jobs, &job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, j := range jobs {
		steps, err := s.listStepsForJob(ctx, j.ID)
		if err == nil {
			j.Steps = steps
		}
	}

	return jobs, nil
}

func (s *sqlRepositoryStore) listStepsForJob(ctx context.Context, jobID string) ([]*JobStep, error) {
	query := `
		SELECT id, pipeline_job_id, step_number, name, command, status,
		       started_at, finished_at, exit_code, created_at
		FROM job_steps
		WHERE pipeline_job_id = $1
		ORDER BY step_number ASC
	`
	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*JobStep
	for rows.Next() {
		var step JobStep
		var statusStr string
		if err := rows.Scan(
			&step.ID, &step.PipelineJobID, &step.StepNumber, &step.Name, &step.Command, &statusStr,
			&step.StartedAt, &step.FinishedAt, &step.ExitCode, &step.CreatedAt,
		); err != nil {
			return nil, err
		}
		step.Status = StepStatus(statusStr)
		if step.StartedAt != nil && step.FinishedAt != nil {
			step.DurationMs = step.FinishedAt.Sub(*step.StartedAt).Milliseconds()
		}
		steps = append(steps, &step)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return steps, nil
}

func (s *sqlRepositoryStore) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, exitCode *int, startedAt, finishedAt *time.Time) error {
	query := `
		UPDATE pipeline_jobs
		SET status = $1, exit_code = $2, started_at = COALESCE($3, started_at), finished_at = $4, updated_at = NOW()
		WHERE id = $5
	`
	_, err := s.db.ExecContext(ctx, query, string(status), exitCode, startedAt, finishedAt, jobID)
	return err
}

func (s *sqlRepositoryStore) CreateJobStep(ctx context.Context, step *JobStep) error {
	if step.ID == "" {
		step.ID = uuid.NewString()
	}
	step.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO job_steps (
			id, pipeline_job_id, step_number, name, command, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		step.ID, step.PipelineJobID, step.StepNumber, step.Name, step.Command, string(step.Status), step.CreatedAt,
	)
	return err
}

func (s *sqlRepositoryStore) UpdateStepStatus(ctx context.Context, stepID string, status StepStatus, exitCode *int, startedAt, finishedAt *time.Time) error {
	query := `
		UPDATE job_steps
		SET status = $1, exit_code = $2, started_at = COALESCE($3, started_at), finished_at = $4
		WHERE id = $5
	`
	_, err := s.db.ExecContext(ctx, query, string(status), exitCode, startedAt, finishedAt, stepID)
	return err
}

func (s *sqlRepositoryStore) AppendJobLog(ctx context.Context, log *JobLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	log.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO job_logs (
			id, pipeline_job_id, step_id, line_number, content, stream, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		log.ID, log.PipelineJobID, log.StepID, log.LineNumber, log.Content, log.Stream, log.CreatedAt,
	)
	return err
}

func (s *sqlRepositoryStore) GetJobLogs(ctx context.Context, jobID string, afterLine int) ([]*JobLog, error) {
	query := `
		SELECT id, pipeline_job_id, step_id, line_number, content, stream, created_at
		FROM job_logs
		WHERE pipeline_job_id = $1 AND line_number > $2
		ORDER BY line_number ASC
	`
	rows, err := s.db.QueryContext(ctx, query, jobID, afterLine)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*JobLog
	for rows.Next() {
		var log JobLog
		var stepID sql.NullString
		if err := rows.Scan(
			&log.ID, &log.PipelineJobID, &stepID, &log.LineNumber, &log.Content, &log.Stream, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		if stepID.Valid {
			log.StepID = &stepID.String
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

// -------------------------------------------------------------------------
// In-Memory Implementation (for Tests & Zero-Dependency Mode)
// -------------------------------------------------------------------------

type memoryRepositoryStore struct {
	mu       sync.RWMutex
	authRepo auth.Repository
	runs     map[string]*PipelineRun
	jobs     map[string]*PipelineJob
	steps    map[string]*JobStep
	logs     map[string][]*JobLog // keyed by pipeline_job_id
}

func NewMemoryRepositoryStore(authRepo auth.Repository) RepositoryStore {
	return &memoryRepositoryStore{
		authRepo: authRepo,
		runs:     make(map[string]*PipelineRun),
		jobs:     make(map[string]*PipelineJob),
		steps:    make(map[string]*JobStep),
		logs:     make(map[string][]*JobLog),
	}
}

func (m *memoryRepositoryStore) CreatePipelineRun(ctx context.Context, run *PipelineRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	run.CreatedAt = now
	run.UpdatedAt = now

	copied := *run
	m.runs[run.ID] = &copied
	return nil
}

func (m *memoryRepositoryStore) GetPipelineRun(ctx context.Context, runID string) (*PipelineRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	run, ok := m.runs[runID]
	if !ok {
		return nil, ErrRunNotFound
	}
	copied := *run
	if copied.TriggerUserID != nil && m.authRepo != nil {
		if u, _ := m.authRepo.GetUserByID(ctx, *copied.TriggerUserID); u != nil {
			pub := u.ToPublic()
			copied.TriggerUser = &pub
		}
	}
	if copied.StartedAt != nil && copied.FinishedAt != nil {
		copied.DurationMs = copied.FinishedAt.Sub(*copied.StartedAt).Milliseconds()
	}
	return &copied, nil
}

func (m *memoryRepositoryStore) ListPipelineRuns(ctx context.Context, repoID string, filter RunFilter) ([]*PipelineRun, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []*PipelineRun
	for _, r := range m.runs {
		if r.RepositoryID != repoID {
			continue
		}
		if filter.Status != "" && filter.Status != "all" && string(r.Status) != filter.Status {
			continue
		}
		if filter.Branch != "" && r.Branch != filter.Branch {
			continue
		}
		if filter.Event != "" && r.TriggerEvent != filter.Event {
			continue
		}
		copied := *r
		if copied.TriggerUserID != nil && m.authRepo != nil {
			if u, _ := m.authRepo.GetUserByID(ctx, *copied.TriggerUserID); u != nil {
				pub := u.ToPublic()
				copied.TriggerUser = &pub
			}
		}
		if copied.StartedAt != nil && copied.FinishedAt != nil {
			copied.DurationMs = copied.FinishedAt.Sub(*copied.StartedAt).Milliseconds()
		}
		matches = append(matches, &copied)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})

	total := len(matches)
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []*PipelineRun{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return matches[start:end], total, nil
}

func (m *memoryRepositoryStore) UpdatePipelineRunStatus(ctx context.Context, runID string, status PipelineStatus, startedAt, finishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	run, ok := m.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	run.Status = status
	if startedAt != nil {
		run.StartedAt = startedAt
	}
	if finishedAt != nil {
		run.FinishedAt = finishedAt
	}
	run.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryRepositoryStore) CreatePipelineJob(ctx context.Context, job *PipelineJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	copied := *job
	m.jobs[job.ID] = &copied
	return nil
}

func (m *memoryRepositoryStore) GetPipelineJob(ctx context.Context, jobID string) (*PipelineJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	copied := *job
	if copied.StartedAt != nil && copied.FinishedAt != nil {
		copied.DurationMs = copied.FinishedAt.Sub(*copied.StartedAt).Milliseconds()
	}

	// Attach steps
	var steps []*JobStep
	for _, s := range m.steps {
		if s.PipelineJobID == jobID {
			sc := *s
			if sc.StartedAt != nil && sc.FinishedAt != nil {
				sc.DurationMs = sc.FinishedAt.Sub(*sc.StartedAt).Milliseconds()
			}
			steps = append(steps, &sc)
		}
	}
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].StepNumber < steps[j].StepNumber
	})
	copied.Steps = steps

	return &copied, nil
}

func (m *memoryRepositoryStore) ListJobsForRun(ctx context.Context, runID string) ([]*PipelineJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*PipelineJob
	for _, j := range m.jobs {
		if j.PipelineRunID == runID {
			jc := *j
			if jc.StartedAt != nil && jc.FinishedAt != nil {
				jc.DurationMs = jc.FinishedAt.Sub(*jc.StartedAt).Milliseconds()
			}

			var steps []*JobStep
			for _, s := range m.steps {
				if s.PipelineJobID == j.ID {
					sc := *s
					if sc.StartedAt != nil && sc.FinishedAt != nil {
						sc.DurationMs = sc.FinishedAt.Sub(*sc.StartedAt).Milliseconds()
					}
					steps = append(steps, &sc)
				}
			}
			sort.Slice(steps, func(a, b int) bool {
				return steps[a].StepNumber < steps[b].StepNumber
			})
			jc.Steps = steps
			jobs = append(jobs, &jc)
		}
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	return jobs, nil
}

func (m *memoryRepositoryStore) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, exitCode *int, startedAt, finishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	job.Status = status
	if exitCode != nil {
		job.ExitCode = exitCode
	}
	if startedAt != nil {
		job.StartedAt = startedAt
	}
	if finishedAt != nil {
		job.FinishedAt = finishedAt
	}
	job.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryRepositoryStore) CreateJobStep(ctx context.Context, step *JobStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if step.ID == "" {
		step.ID = uuid.NewString()
	}
	step.CreatedAt = time.Now().UTC()
	copied := *step
	m.steps[step.ID] = &copied
	return nil
}

func (m *memoryRepositoryStore) UpdateStepStatus(ctx context.Context, stepID string, status StepStatus, exitCode *int, startedAt, finishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	step, ok := m.steps[stepID]
	if !ok {
		return fmt.Errorf("step not found")
	}
	step.Status = status
	if exitCode != nil {
		step.ExitCode = exitCode
	}
	if startedAt != nil {
		step.StartedAt = startedAt
	}
	if finishedAt != nil {
		step.FinishedAt = finishedAt
	}
	return nil
}

func (m *memoryRepositoryStore) AppendJobLog(ctx context.Context, log *JobLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	log.CreatedAt = time.Now().UTC()
	copied := *log
	m.logs[log.PipelineJobID] = append(m.logs[log.PipelineJobID], &copied)
	return nil
}

func (m *memoryRepositoryStore) GetJobLogs(ctx context.Context, jobID string, afterLine int) ([]*JobLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allLogs := m.logs[jobID]
	var out []*JobLog
	for _, l := range allLogs {
		if l.LineNumber > afterLine {
			copied := *l
			out = append(out, &copied)
		}
	}
	return out, nil
}

package ci

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Executor interface {
	ExecuteJob(ctx context.Context, run *PipelineRun, job *PipelineJob, repoOwner, repoSlug, workspaceDir string) error
	CancelJob(jobID string)
}

type localExecutor struct {
	store     RepositoryStore
	cancelsMu sync.Mutex
	cancels   map[string]context.CancelFunc
}

func NewLocalExecutor(store RepositoryStore) Executor {
	return &localExecutor{
		store:   store,
		cancels: make(map[string]context.CancelFunc),
	}
}

func (e *localExecutor) CancelJob(jobID string) {
	e.cancelsMu.Lock()
	defer e.cancelsMu.Unlock()
	if cancel, ok := e.cancels[jobID]; ok {
		cancel()
		delete(e.cancels, jobID)
	}
}

func (e *localExecutor) ExecuteJob(ctx context.Context, run *PipelineRun, job *PipelineJob, repoOwner, repoSlug, workspaceDir string) error {
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	e.cancelsMu.Lock()
	e.cancels[job.ID] = cancel
	e.cancelsMu.Unlock()

	defer func() {
		e.cancelsMu.Lock()
		delete(e.cancels, job.ID)
		e.cancelsMu.Unlock()
		cancel()
	}()

	now := time.Now().UTC()
	_ = e.store.UpdateJobStatus(ctx, job.ID, JobRunning, nil, &now, nil)

	// Fetch fresh job details with steps
	freshJob, err := e.store.GetPipelineJob(ctx, job.ID)
	if err != nil {
		nowFinish := time.Now().UTC()
		ec := 1
		_ = e.store.UpdateJobStatus(ctx, job.ID, JobFailure, &ec, &now, &nowFinish)
		return err
	}

	var lineCounter int64
	jobFailed := false
	var finalExitCode int

	// Base environment variables
	baseEnv := []string{
		"CI=true",
		"FORGE_CI=true",
		fmt.Sprintf("FORGE_REPOSITORY=%s/%s", repoOwner, repoSlug),
		fmt.Sprintf("FORGE_REF=%s", run.Branch),
		fmt.Sprintf("FORGE_SHA=%s", run.CommitSHA),
		fmt.Sprintf("FORGE_RUN_ID=%s", run.ID),
		fmt.Sprintf("FORGE_JOB_ID=%s", job.ID),
		fmt.Sprintf("FORGE_WORKSPACE=%s", workspaceDir),
	}

	// Iterate steps sequentially
	for _, step := range freshJob.Steps {
		if jobCtx.Err() != nil {
			// Cancelled
			_ = e.store.UpdateStepStatus(ctx, step.ID, StepSkipped, nil, nil, nil)
			continue
		}

		if jobFailed {
			_ = e.store.UpdateStepStatus(ctx, step.ID, StepSkipped, nil, nil, nil)
			continue
		}

		stepStart := time.Now().UTC()
		_ = e.store.UpdateStepStatus(ctx, step.ID, StepRunning, nil, &stepStart, nil)

		// System header log
		lineNum := atomic.AddInt64(&lineCounter, 1)
		_ = e.store.AppendJobLog(ctx, &JobLog{
			PipelineJobID: job.ID,
			StepID:        &step.ID,
			LineNumber:    int(lineNum),
			Content:       fmt.Sprintf("##[group]Step %d: %s", step.StepNumber, step.Name),
			Stream:        "system",
		})

		stepExitCode, err := e.runCommand(jobCtx, step.Command, workspaceDir, baseEnv, job.ID, step.ID, &lineCounter)

		stepFinish := time.Now().UTC()
		if err != nil || stepExitCode != 0 {
			jobFailed = true
			finalExitCode = stepExitCode
			if finalExitCode == 0 {
				finalExitCode = 1
			}

			_ = e.store.UpdateStepStatus(ctx, step.ID, StepFailure, &finalExitCode, &stepStart, &stepFinish)

			lineNum = atomic.AddInt64(&lineCounter, 1)
			_ = e.store.AppendJobLog(ctx, &JobLog{
				PipelineJobID: job.ID,
				StepID:        &step.ID,
				LineNumber:    int(lineNum),
				Content:       fmt.Sprintf("##[error]Process completed with exit code %d.", finalExitCode),
				Stream:        "stderr",
			})
		} else {
			_ = e.store.UpdateStepStatus(ctx, step.ID, StepSuccess, &stepExitCode, &stepStart, &stepFinish)
		}

		lineNum = atomic.AddInt64(&lineCounter, 1)
		_ = e.store.AppendJobLog(ctx, &JobLog{
			PipelineJobID: job.ID,
			StepID:        &step.ID,
			LineNumber:    int(lineNum),
			Content:       "##[endgroup]",
			Stream:        "system",
		})
	}

	jobFinish := time.Now().UTC()
	var finalJobStatus JobStatus
	if jobCtx.Err() == context.Canceled {
		finalJobStatus = JobCancelled
	} else if jobFailed {
		finalJobStatus = JobFailure
	} else {
		finalJobStatus = JobSuccess
		finalExitCode = 0
	}

	_ = e.store.UpdateJobStatus(ctx, job.ID, finalJobStatus, &finalExitCode, &now, &jobFinish)

	// Check if all jobs in this run are finished to update run status
	e.evaluateRunCompletion(ctx, run.ID)
	return nil
}

func (e *localExecutor) runCommand(ctx context.Context, command, workspaceDir string, env []string, jobID, stepID string, lineCounter *int64) (int, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return 0, nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", trimmed)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", trimmed)
	}

	if workspaceDir != "" {
		cmd.Dir = workspaceDir
	}
	cmd.Env = append(os.Environ(), env...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 1, err
	}

	if err := cmd.Start(); err != nil {
		lineNum := atomic.AddInt64(lineCounter, 1)
		_ = e.store.AppendJobLog(ctx, &JobLog{
			PipelineJobID: jobID,
			StepID:        &stepID,
			LineNumber:    int(lineNum),
			Content:       fmt.Sprintf("Failed to start command: %v", err),
			Stream:        "stderr",
		})
		return 1, err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stdout
	go func() {
		defer wg.Done()
		e.streamPipe(stdoutPipe, jobID, stepID, "stdout", lineCounter)
	}()

	// Stream stderr
	go func() {
		defer wg.Done()
		e.streamPipe(stderrPipe, jobID, stepID, "stderr", lineCounter)
	}()

	wg.Wait()

	waitErr := cmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, waitErr
	}

	return 0, nil
}

func (e *localExecutor) streamPipe(r io.Reader, jobID, stepID, stream string, lineCounter *int64) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum := atomic.AddInt64(lineCounter, 1)
		_ = e.store.AppendJobLog(context.Background(), &JobLog{
			PipelineJobID: jobID,
			StepID:        &stepID,
			LineNumber:    int(lineNum),
			Content:       line,
			Stream:        stream,
		})
	}
}

func (e *localExecutor) evaluateRunCompletion(ctx context.Context, runID string) {
	jobs, err := e.store.ListJobsForRun(ctx, runID)
	if err != nil || len(jobs) == 0 {
		return
	}

	allFinished := true
	hasFailure := false
	hasCancelled := false

	for _, j := range jobs {
		switch j.Status {
		case JobQueued, JobRunning:
			allFinished = false
		case JobFailure:
			hasFailure = true
		case JobCancelled:
			hasCancelled = true
		}
	}

	if allFinished {
		now := time.Now().UTC()
		var finalStatus PipelineStatus
		if hasCancelled {
			finalStatus = StatusCancelled
		} else if hasFailure {
			finalStatus = StatusFailure
		} else {
			finalStatus = StatusSuccess
		}
		_ = e.store.UpdatePipelineRunStatus(ctx, runID, finalStatus, nil, &now)
	}
}

// Helper to prepare an execution workspace directory
func PrepareWorkspace(tempBase, owner, repo, branch string) (string, func(), error) {
	dir, err := os.MkdirTemp(tempBase, fmt.Sprintf("forge-run-%s-%s-*", owner, repo))
	if err != nil {
		return "", func() {}, err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", func() {}, err
	}

	cleanup := func() {
		_ = os.RemoveAll(abs)
	}
	return abs, cleanup, nil
}

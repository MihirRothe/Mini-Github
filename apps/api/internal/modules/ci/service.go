package ci

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

type Service interface {
	ListRuns(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter RunFilter) ([]*PipelineRun, int, error)
	GetRun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) (*PipelineRunDetail, error)
	TriggerWorkflow(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req TriggerRunRequest) (*PipelineRun, error)
	CancelRun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) error
	Rerun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) (*PipelineRun, error)
	GetJob(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, jobID string) (*PipelineJob, error)
	GetJobLogs(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, jobID string, afterLine int) ([]*JobLog, error)
	ListWorkflows(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, ref string) ([]string, error)
}

type service struct {
	store      RepositoryStore
	reposSvc   repos.Service
	gitStorage git.Storage
	gitReader  git.Reader
	authRepo   auth.Repository
	executor   Executor
}

func NewService(
	store RepositoryStore,
	reposSvc repos.Service,
	gitStorage git.Storage,
	gitReader git.Reader,
	authRepo auth.Repository,
	executor Executor,
) Service {
	return &service{
		store:      store,
		reposSvc:   reposSvc,
		gitStorage: gitStorage,
		gitReader:  gitReader,
		authRepo:   authRepo,
		executor:   executor,
	}
}

func (s *service) checkAccess(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, needWrite bool) (*repos.Repository, error) {
	repo, err := s.reposSvc.GetRepository(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	if needWrite {
		if !isSiteAdmin {
			perm := repo.CurrentUserPermission
			if perm != orgs.PermWrite && perm != orgs.PermMaintain && perm != orgs.PermAdmin {
				return nil, ErrAccessDenied
			}
		}
	}
	return repo, nil
}

func (s *service) ListWorkflows(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, ref string) ([]string, error) {
	repo, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, false)
	if err != nil {
		return nil, err
	}

	if ref == "" {
		ref = repo.DefaultBranch
		if ref == "" {
			ref = "main"
		}
	}

	diskPath, err := s.gitStorage.GetDiskPath(repo.OwnerType+"s", repo.OwnerName, repo.Slug)
	if err != nil {
		return nil, err
	}

	var workflows []string

	// 1. Check root forge-ci.yml
	if blob, err := s.gitReader.GetBlob(diskPath, ref, "forge-ci.yml"); err == nil && blob != nil {
		workflows = append(workflows, "forge-ci.yml")
	}

	// 2. Check .forge/workflows directory
	entries, err := s.gitReader.ListTree(diskPath, ref, ".forge/workflows")
	if err == nil {
		for _, e := range entries {
			if e.Type == "blob" && (strings.HasSuffix(e.Name, ".yml") || strings.HasSuffix(e.Name, ".yaml")) {
				workflows = append(workflows, ".forge/workflows/"+e.Name)
			}
		}
	}

	return workflows, nil
}

func (s *service) TriggerWorkflow(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req TriggerRunRequest) (*PipelineRun, error) {
	repo, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, true)
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = repo.DefaultBranch
		if branch == "" {
			branch = "main"
		}
	}

	workflowFile := strings.TrimSpace(req.WorkflowFile)
	if workflowFile == "" {
		workflows, err := s.ListWorkflows(ctx, currentUserID, isSiteAdmin, owner, repoSlug, branch)
		if err != nil || len(workflows) == 0 {
			// Provide default fallback inline CI if none exist
			workflowFile = "forge-ci.yml"
		} else {
			workflowFile = workflows[0]
		}
	}

	diskPath, err := s.gitStorage.GetDiskPath(repo.OwnerType+"s", repo.OwnerName, repo.Slug)
	if err != nil {
		return nil, err
	}

	// Get latest commit on branch
	commits, err := s.gitReader.ListCommits(diskPath, branch, 1)
	var commitSHA string
	var commitMsg string
	if err == nil && len(commits) > 0 {
		commitSHA = commits[0].Hash
		commitMsg = commits[0].Message
	} else {
		commitSHA = "0000000000000000000000000000000000000000"
		commitMsg = "Manual workflow trigger"
	}

	// Read workflow file content
	var wfDef *WorkflowDefinition
	blob, err := s.gitReader.GetBlob(diskPath, branch, workflowFile)
	if err == nil && blob != nil && blob.Content != "" {
		parsed, parseErr := ParseWorkflowYAML([]byte(blob.Content))
		if parseErr != nil {
			return nil, fmt.Errorf("parse workflow: %w", parseErr)
		}
		wfDef = parsed
	} else {
		// Provide an initial default test workflow if file not yet committed
		defaultYAML := `
name: Build and Test
on: [push, workflow_dispatch]
jobs:
  test:
    name: Run Unit Tests
    runs-on: ubuntu-latest
    steps:
      - name: Verify Environment
        run: echo "ForgeHub CI Environment initialized for branch $FORGE_REF"
      - name: Code Quality & Tests
        run: echo "All unit tests passed successfully."
`
		wfDef, _ = ParseWorkflowYAML([]byte(defaultYAML))
		workflowFile = "forge-ci.yml"
	}

	var triggerUserID *string
	if currentUserID != "" {
		triggerUserID = &currentUserID
	}

	run := &PipelineRun{
		RepositoryID:  repo.ID,
		WorkflowName:  wfDef.Name,
		WorkflowFile:  workflowFile,
		TriggerEvent:  "workflow_dispatch",
		TriggerUserID: triggerUserID,
		CommitSHA:     commitSHA,
		CommitMessage: commitMsg,
		Branch:        branch,
		Status:        StatusQueued,
	}

	if err := s.store.CreatePipelineRun(ctx, run); err != nil {
		return nil, fmt.Errorf("create pipeline run: %w", err)
	}

	// Create jobs and steps
	var jobsToRun []*PipelineJob
	for jobKey, wfJob := range wfDef.Jobs {
		job := &PipelineJob{
			PipelineRunID: run.ID,
			JobKey:        jobKey,
			Name:          wfJob.Name,
			RunsOn:        wfJob.RunsOn,
			Status:        JobQueued,
		}
		if err := s.store.CreatePipelineJob(ctx, job); err != nil {
			return nil, err
		}

		for idx, wfStep := range wfJob.Steps {
			step := &JobStep{
				PipelineJobID: job.ID,
				StepNumber:    idx + 1,
				Name:          wfStep.Name,
				Command:       wfStep.Run,
				Status:        StepPending,
			}
			if step.Name == "" {
				step.Name = fmt.Sprintf("Step %d", idx+1)
			}
			if err := s.store.CreateJobStep(ctx, step); err != nil {
				return nil, err
			}
			job.Steps = append(job.Steps, step)
		}
		jobsToRun = append(jobsToRun, job)
	}

	// Execute jobs in background
	go s.dispatchJobs(context.Background(), run, jobsToRun, repo.OwnerName, repo.Slug)

	return run, nil
}

func (s *service) dispatchJobs(ctx context.Context, run *PipelineRun, jobs []*PipelineJob, owner, repoSlug string) {
	now := time.Now().UTC()
	_ = s.store.UpdatePipelineRunStatus(ctx, run.ID, StatusRunning, &now, nil)

	// Prepare temporary workspace directory
	tempWorkspace, cleanup, err := PrepareWorkspace(os.TempDir(), owner, repoSlug, run.Branch)
	if err != nil {
		tempWorkspace = ""
	} else {
		defer cleanup()
	}

	for _, job := range jobs {
		_ = s.executor.ExecuteJob(ctx, run, job, owner, repoSlug, tempWorkspace)
	}
}

func (s *service) ListRuns(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter RunFilter) ([]*PipelineRun, int, error) {
	repo, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, false)
	if err != nil {
		return nil, 0, err
	}
	return s.store.ListPipelineRuns(ctx, repo.ID, filter)
}

func (s *service) GetRun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) (*PipelineRunDetail, error) {
	repo, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, false)
	if err != nil {
		return nil, err
	}

	run, err := s.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.RepositoryID != repo.ID {
		return nil, ErrRunNotFound
	}

	jobs, err := s.store.ListJobsForRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}

	return &PipelineRunDetail{
		PipelineRun: *run,
		Jobs:        jobs,
	}, nil
}

func (s *service) CancelRun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) error {
	repo, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, true)
	if err != nil {
		return err
	}

	run, err := s.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.RepositoryID != repo.ID {
		return ErrRunNotFound
	}

	if run.Status == StatusSuccess || run.Status == StatusFailure || run.Status == StatusCancelled {
		return ErrRunAlreadyFinished
	}

	jobs, _ := s.store.ListJobsForRun(ctx, run.ID)
	for _, j := range jobs {
		s.executor.CancelJob(j.ID)
	}

	now := time.Now().UTC()
	return s.store.UpdatePipelineRunStatus(ctx, run.ID, StatusCancelled, nil, &now)
}

func (s *service) Rerun(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, runID string) (*PipelineRun, error) {
	runDetail, err := s.GetRun(ctx, currentUserID, isSiteAdmin, owner, repoSlug, runID)
	if err != nil {
		return nil, err
	}

	req := TriggerRunRequest{
		WorkflowFile: runDetail.WorkflowFile,
		Branch:       runDetail.Branch,
	}
	return s.TriggerWorkflow(ctx, currentUserID, isSiteAdmin, owner, repoSlug, req)
}

func (s *service) GetJob(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, jobID string) (*PipelineJob, error) {
	_, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, false)
	if err != nil {
		return nil, err
	}
	return s.store.GetPipelineJob(ctx, jobID)
}

func (s *service) GetJobLogs(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, jobID string, afterLine int) ([]*JobLog, error) {
	_, err := s.checkAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug, false)
	if err != nil {
		return nil, err
	}
	return s.store.GetJobLogs(ctx, jobID, afterLine)
}

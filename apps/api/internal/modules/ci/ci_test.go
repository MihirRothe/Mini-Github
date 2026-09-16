package ci

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

func TestWorkflowYAMLParser(t *testing.T) {
	yamlContent := `
name: Build and Test Matrix
on:
  push:
    branches: [main, "release/*"]
  pull_request:
    branches: [main]
  workflow_dispatch:
env:
  NODE_ENV: test
jobs:
  build:
    name: Build Binary
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        run: echo "Checking out repository..."
      - name: Compile
        run: echo "Compiling binary..."
        env:
          CGO_ENABLED: "0"
`
	def, err := ParseWorkflowYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("unexpected error parsing workflow YAML: %v", err)
	}

	if def.Name != "Build and Test Matrix" {
		t.Errorf("expected name 'Build and Test Matrix', got '%s'", def.Name)
	}
	if !def.On.MatchesPush("main") {
		t.Errorf("expected push to match 'main'")
	}
	if !def.On.MatchesPush("release/1.0") {
		t.Errorf("expected push to match 'release/1.0'")
	}
	if def.On.MatchesPush("feature/auth") {
		t.Errorf("did not expect push to match 'feature/auth'")
	}
	if !def.On.MatchesPullRequest("main") {
		t.Errorf("expected pull_request to match 'main'")
	}
	if !def.On.MatchesDispatch() {
		t.Errorf("expected workflow_dispatch to match")
	}

	job, ok := def.Jobs["build"]
	if !ok {
		t.Fatalf("expected job 'build' in parsed workflow")
	}
	if job.Name != "Build Binary" {
		t.Errorf("expected job name 'Build Binary', got '%s'", job.Name)
	}
	if len(job.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(job.Steps))
	}
	if job.Steps[1].Name != "Compile" {
		t.Errorf("expected second step name 'Compile', got '%s'", job.Steps[1].Name)
	}
}

func setupCITest(t *testing.T) (Service, RepositoryStore, repos.Service, *auth.Service, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "forgehub-ci-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cfg := &config.Config{
		GitRootDir:       tempDir,
		GitDefaultBranch: "main",
		BaseURL:          "http://localhost:8080",
	}

	authRepo := auth.NewMemoryRepository()
	authSvc := auth.NewService(authRepo, cfg)
	orgsRepo := orgs.NewMemoryRepository(authRepo)
	repoStore := repos.NewMemoryRepositoryStore()

	gitStorage, err := git.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("create git storage: %v", err)
	}
	gitReader := git.NewReader()

	reposSvc := repos.NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, cfg)

	ciStore := NewMemoryRepositoryStore(authRepo)
	ciExecutor := NewLocalExecutor(ciStore)
	ciSvc := NewService(ciStore, reposSvc, gitStorage, gitReader, authRepo, ciExecutor)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return ciSvc, ciStore, reposSvc, authSvc, cleanup
}

func TestPipelineSuccessfulExecution(t *testing.T) {
	ctx := context.Background()
	ciSvc, ciStore, reposSvc, authSvc, cleanup := setupCITest(t)
	defer cleanup()

	// 1. Create User & Repo
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "ci-dev",
		Email:    "cidev@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	_, err = reposSvc.CreateRepository(ctx, user.ID, false, repos.CreateRepoRequest{
		Name:           "Pipeline Test Repo",
		Slug:           "pipeline-test-repo",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 2. Trigger Workflow
	run, err := ciSvc.TriggerWorkflow(ctx, user.ID, false, "ci-dev", "pipeline-test-repo", TriggerRunRequest{
		Branch: "main",
	})
	if err != nil {
		t.Fatalf("trigger workflow: %v", err)
	}

	if run.Status != StatusQueued && run.Status != StatusRunning {
		t.Errorf("expected initial status queued or running, got %s", run.Status)
	}

	// 3. Wait for pipeline run to complete (up to 10 seconds)
	var finalRun *PipelineRun
	for i := 0; i < 50; i++ {
		time.Sleep(200 * time.Millisecond)
		fresh, err := ciStore.GetPipelineRun(ctx, run.ID)
		if err == nil && (fresh.Status == StatusSuccess || fresh.Status == StatusFailure || fresh.Status == StatusCancelled) {
			finalRun = fresh
			break
		}
	}

	if finalRun == nil {
		t.Fatalf("timed out waiting for pipeline run to finish")
	}

	if finalRun.Status != StatusSuccess {
		t.Errorf("expected pipeline run status success, got %s", finalRun.Status)
	}

	// 4. Verify Job & Steps
	jobs, err := ciStore.ListJobsForRun(ctx, run.ID)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("expected jobs for run, got %v", err)
	}

	job := jobs[0]
	if job.Status != JobSuccess {
		t.Errorf("expected job status success, got %s", job.Status)
	}
	if len(job.Steps) == 0 {
		t.Fatalf("expected job steps to be populated")
	}

	// 5. Verify Execution Logs
	logs, err := ciStore.GetJobLogs(ctx, job.ID, 0)
	if err != nil {
		t.Fatalf("get job logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected non-empty execution logs")
	}

	foundOutput := false
	for _, l := range logs {
		if strings.Contains(l.Content, "ForgeHub CI Environment") || strings.Contains(l.Content, "passed successfully") {
			foundOutput = true
			break
		}
	}
	if !foundOutput {
		t.Errorf("expected expected log content in job logs, got %+v", logs)
	}
}

func TestPipelineFailingExecution(t *testing.T) {
	ctx := context.Background()
	_, ciStore, reposSvc, authSvc, cleanup := setupCITest(t)
	defer cleanup()

	// 1. Create User & Repo
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "fail-dev",
		Email:    "faildev@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	repo, err := reposSvc.CreateRepository(ctx, user.ID, false, repos.CreateRepoRequest{
		Name:           "Fail Pipeline Repo",
		Slug:           "fail-pipeline-repo",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 2. Create Run and Job with failing step manually in store
	run := &PipelineRun{
		RepositoryID: repo.ID,
		WorkflowName: "Test Failing Pipeline",
		WorkflowFile: "forge-ci.yml",
		TriggerEvent: "workflow_dispatch",
		CommitSHA:    "abc123",
		Branch:       "main",
		Status:       StatusRunning,
	}
	_ = ciStore.CreatePipelineRun(ctx, run)

	job := &PipelineJob{
		PipelineRunID: run.ID,
		JobKey:        "test",
		Name:          "Failing Job",
		RunsOn:        "ubuntu-latest",
		Status:        JobQueued,
	}
	_ = ciStore.CreatePipelineJob(ctx, job)

	step1 := &JobStep{
		PipelineJobID: job.ID,
		StepNumber:    1,
		Name:          "Pass Step",
		Command:       "echo \"Step 1 OK\"",
		Status:        StepPending,
	}
	_ = ciStore.CreateJobStep(ctx, step1)

	step2 := &JobStep{
		PipelineJobID: job.ID,
		StepNumber:    2,
		Name:          "Fail Step",
		// Command that exits with non-zero
		Command: "powershell -Command \"exit 42\"",
		Status:  StepPending,
	}
	_ = ciStore.CreateJobStep(ctx, step2)

	step3 := &JobStep{
		PipelineJobID: job.ID,
		StepNumber:    3,
		Name:          "Skipped Step",
		Command:       "echo \"Should not run\"",
		Status:        StepPending,
	}
	_ = ciStore.CreateJobStep(ctx, step3)

	// 3. Execute Job
	executor := NewLocalExecutor(ciStore)
	err = executor.ExecuteJob(ctx, run, job, "fail-dev", "fail-pipeline-repo", "")
	if err != nil {
		t.Fatalf("execute job: %v", err)
	}

	// 4. Verify Final Statuses
	freshJob, _ := ciStore.GetPipelineJob(ctx, job.ID)
	if freshJob.Status != JobFailure {
		t.Errorf("expected job status failure, got %s", freshJob.Status)
	}

	if len(freshJob.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(freshJob.Steps))
	}

	if freshJob.Steps[0].Status != StepSuccess {
		t.Errorf("step 1 should be success, got %s", freshJob.Steps[0].Status)
	}
	if freshJob.Steps[1].Status != StepFailure {
		t.Errorf("step 2 should be failure, got %s", freshJob.Steps[1].Status)
	}
	if freshJob.Steps[2].Status != StepSkipped {
		t.Errorf("step 3 should be skipped, got %s", freshJob.Steps[2].Status)
	}

	freshRun, _ := ciStore.GetPipelineRun(ctx, run.ID)
	if freshRun.Status != StatusFailure {
		t.Errorf("expected run status failure, got %s", freshRun.Status)
	}
}

package ci

import (
	"errors"
	"time"

	"forgehub/apps/api/internal/modules/auth"
)

var (
	ErrWorkflowNotFound   = errors.New("workflow file not found in repository")
	ErrInvalidWorkflowDef = errors.New("invalid workflow YAML definition")
	ErrRunNotFound        = errors.New("pipeline run not found")
	ErrJobNotFound        = errors.New("pipeline job not found")
	ErrRunAlreadyFinished = errors.New("pipeline run is already finished")
	ErrAccessDenied       = errors.New("access denied to pipeline action")
)

type PipelineStatus string

const (
	StatusQueued    PipelineStatus = "queued"
	StatusRunning   PipelineStatus = "running"
	StatusSuccess   PipelineStatus = "success"
	StatusFailure   PipelineStatus = "failure"
	StatusCancelled PipelineStatus = "cancelled"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobSuccess   JobStatus = "success"
	JobFailure   JobStatus = "failure"
	JobSkipped   JobStatus = "skipped"
	JobCancelled JobStatus = "cancelled"
)

type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepSuccess StepStatus = "success"
	StepFailure StepStatus = "failure"
	StepSkipped StepStatus = "skipped"
)

// Workflow YAML Structure (.forge/workflows/*.yml)
type WorkflowDefinition struct {
	Name string                 `yaml:"name"`
	On   WorkflowTriggers       `yaml:"on"`
	Env  map[string]string      `yaml:"env,omitempty"`
	Jobs map[string]WorkflowJob `yaml:"jobs"`
}

type WorkflowTriggers struct {
	Push             *TriggerEvent   `yaml:"push,omitempty"`
	PullRequest      *TriggerEvent   `yaml:"pull_request,omitempty"`
	WorkflowDispatch *struct{}       `yaml:"workflow_dispatch,omitempty"`
	RawStrings       []string        `yaml:"-"`
}

type TriggerEvent struct {
	Branches []string `yaml:"branches,omitempty"`
	Paths    []string `yaml:"paths,omitempty"`
}

type WorkflowJob struct {
	Name   string            `yaml:"name"`
	RunsOn string            `yaml:"runs-on"`
	Env    map[string]string `yaml:"env,omitempty"`
	Steps  []WorkflowStep    `yaml:"steps"`
}

type WorkflowStep struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run"`
	Uses string            `yaml:"uses,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
}

// Domain Models

type PipelineRun struct {
	ID            string           `json:"id"`
	RepositoryID  string           `json:"repository_id"`
	WorkflowName  string           `json:"workflow_name"`
	WorkflowFile  string           `json:"workflow_file"`
	TriggerEvent  string           `json:"trigger_event"`
	TriggerUserID *string          `json:"trigger_user_id,omitempty"`
	TriggerUser   *auth.PublicUser `json:"trigger_user,omitempty"`
	CommitSHA     string           `json:"commit_sha"`
	CommitMessage string           `json:"commit_message"`
	Branch        string           `json:"branch"`
	Status        PipelineStatus   `json:"status"`
	StartedAt     *time.Time       `json:"started_at,omitempty"`
	FinishedAt    *time.Time       `json:"finished_at,omitempty"`
	DurationMs    int64            `json:"duration_ms"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type PipelineJob struct {
	ID            string     `json:"id"`
	PipelineRunID string     `json:"pipeline_run_id"`
	JobKey        string     `json:"job_key"`
	Name          string     `json:"name"`
	RunsOn        string     `json:"runs-on"`
	Status        JobStatus  `json:"status"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	DurationMs    int64      `json:"duration_ms"`
	ExitCode      *int       `json:"exit_code,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Steps         []*JobStep `json:"steps,omitempty"`
}

type JobStep struct {
	ID            string     `json:"id"`
	PipelineJobID string     `json:"pipeline_job_id"`
	StepNumber    int        `json:"step_number"`
	Name          string     `json:"name"`
	Command       string     `json:"command"`
	Status        StepStatus `json:"status"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	DurationMs    int64      `json:"duration_ms"`
	ExitCode      *int       `json:"exit_code,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type JobLog struct {
	ID            string    `json:"id"`
	PipelineJobID string    `json:"pipeline_job_id"`
	StepID        *string   `json:"step_id,omitempty"`
	LineNumber    int       `json:"line_number"`
	Content       string    `json:"content"`
	Stream        string    `json:"stream"` // stdout, stderr, system
	CreatedAt     time.Time `json:"created_at"`
}

type PipelineRunDetail struct {
	PipelineRun
	Jobs []*PipelineJob `json:"jobs"`
}

// Request & Filter DTOs

type TriggerRunRequest struct {
	WorkflowFile string `json:"workflow_file"`
	Branch       string `json:"branch"`
}

type RunFilter struct {
	Status   string `json:"status"`
	Branch   string `json:"branch"`
	Event    string `json:"event"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

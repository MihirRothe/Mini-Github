package ci

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML custom unmarshaler for WorkflowTriggers to support string, []string, and map
func (wt *WorkflowTriggers) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// e.g. "on: push"
		var single string
		if err := value.Decode(&single); err != nil {
			return err
		}
		wt.RawStrings = []string{single}
		wt.applyString(single)
		return nil

	case yaml.SequenceNode:
		// e.g. "on: [push, pull_request, workflow_dispatch]"
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		wt.RawStrings = list
		for _, s := range list {
			wt.applyString(s)
		}
		return nil

	case yaml.MappingNode:
		// Iterate key-value pairs in MappingNode
		for i := 0; i < len(value.Content); i += 2 {
			k := strings.ToLower(strings.TrimSpace(value.Content[i].Value))
			val := value.Content[i+1]

			switch k {
			case "push":
				var p TriggerEvent
				_ = val.Decode(&p)
				wt.Push = &p
			case "pull_request":
				var pr TriggerEvent
				_ = val.Decode(&pr)
				wt.PullRequest = &pr
			case "workflow_dispatch", "manual":
				wt.WorkflowDispatch = &struct{}{}
			}
		}
		return nil

	default:
		return fmt.Errorf("unexpected node kind for 'on': %v", value.Kind)
	}
}

func (wt *WorkflowTriggers) applyString(s string) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "push":
		if wt.Push == nil {
			wt.Push = &TriggerEvent{}
		}
	case "pull_request":
		if wt.PullRequest == nil {
			wt.PullRequest = &TriggerEvent{}
		}
	case "workflow_dispatch", "manual":
		wt.WorkflowDispatch = &struct{}{}
	}
}

func matchBranch(pattern, branch string) bool {
	if pattern == branch || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(branch, prefix+"/")
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(branch, prefix)
	}
	return false
}

// MatchesPush returns true if the trigger allows a push event to this branch
func (wt *WorkflowTriggers) MatchesPush(branch string) bool {
	if wt.Push == nil {
		return false
	}
	if len(wt.Push.Branches) == 0 {
		return true // triggers on all branches if none specified
	}
	for _, b := range wt.Push.Branches {
		if matchBranch(b, branch) {
			return true
		}
	}
	return false
}

// MatchesPullRequest returns true if the trigger allows a PR event to this branch
func (wt *WorkflowTriggers) MatchesPullRequest(targetBranch string) bool {
	if wt.PullRequest == nil {
		return false
	}
	if len(wt.PullRequest.Branches) == 0 {
		return true
	}
	for _, b := range wt.PullRequest.Branches {
		if matchBranch(b, targetBranch) {
			return true
		}
	}
	return false
}

// MatchesDispatch returns true if manual dispatch is allowed
func (wt *WorkflowTriggers) MatchesDispatch() bool {
	return wt.WorkflowDispatch != nil
}

// ParseWorkflowYAML parses and validates a workflow YAML file
func ParseWorkflowYAML(content []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := yaml.Unmarshal(content, &def); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWorkflowDef, err)
	}

	if strings.TrimSpace(def.Name) == "" {
		def.Name = "Continuous Integration"
	}

	if len(def.Jobs) == 0 {
		return nil, fmt.Errorf("%w: workflow contains no jobs", ErrInvalidWorkflowDef)
	}

	for key, job := range def.Jobs {
		if len(job.Steps) == 0 {
			return nil, fmt.Errorf("%w: job '%s' has no steps", ErrInvalidWorkflowDef, key)
		}
		if job.RunsOn == "" {
			job.RunsOn = "ubuntu-latest"
		}
		if job.Name == "" {
			job.Name = key
		}
		def.Jobs[key] = job
	}

	return &def, nil
}

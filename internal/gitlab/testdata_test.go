package gitlab

import (
	"testing"
)

func TestLoadTestGroups(t *testing.T) {
	groups, err := LoadTestGroups()
	if err != nil {
		t.Fatalf("failed to load groups: %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	if groups[0].Name != "Acme Corp" {
		t.Errorf("expected first group 'Acme Corp', got '%s'", groups[0].Name)
	}

	if groups[1].ParentID == nil || *groups[1].ParentID != 1001 {
		t.Error("expected Backend group to have parent_id 1001")
	}
}

func TestLoadTestProjects(t *testing.T) {
	projects, err := LoadTestProjects()
	if err != nil {
		t.Fatalf("failed to load projects: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(projects))
	}

	if projects[0].Name != "camera-service" {
		t.Errorf("expected first project 'camera-service', got '%s'", projects[0].Name)
	}

	if projects[0].Namespace == nil {
		t.Error("expected project to have namespace")
	}
}

func TestLoadTestPipelines(t *testing.T) {
	pipelines, err := LoadTestPipelines()
	if err != nil {
		t.Fatalf("failed to load pipelines: %v", err)
	}

	if len(pipelines) != 3 {
		t.Errorf("expected 3 pipelines, got %d", len(pipelines))
	}

	statuses := map[string]bool{"success": false, "running": false, "failed": false}
	for _, p := range pipelines {
		statuses[p.Status] = true
	}

	for status, found := range statuses {
		if !found {
			t.Errorf("expected to find pipeline with status '%s'", status)
		}
	}
}

func TestLoadTestMergeRequests(t *testing.T) {
	mrs, err := LoadTestMergeRequests()
	if err != nil {
		t.Fatalf("failed to load merge requests: %v", err)
	}

	if len(mrs) != 3 {
		t.Errorf("expected 3 merge requests, got %d", len(mrs))
	}

	hasDraft := false
	for _, mr := range mrs {
		if mr.Draft {
			hasDraft = true
			break
		}
	}
	if !hasDraft {
		t.Error("expected at least one draft MR")
	}
}

func TestLoadTestBranches(t *testing.T) {
	branches, err := LoadTestBranches()
	if err != nil {
		t.Fatalf("failed to load branches: %v", err)
	}

	if len(branches) != 3 {
		t.Errorf("expected 3 branches, got %d", len(branches))
	}

	for _, b := range branches {
		if b.Name == "main" {
			if !b.Default {
				t.Error("expected main to be default branch")
			}
			if !b.Protected {
				t.Error("expected main to be protected")
			}
		}
	}
}

func TestLoadTestTree(t *testing.T) {
	entries, err := LoadTestTree()
	if err != nil {
		t.Fatalf("failed to load tree: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one tree entry")
	}

	hasDir := false
	hasFile := false
	for _, e := range entries {
		if e.Type == "tree" {
			hasDir = true
		}
		if e.Type == "blob" {
			hasFile = true
		}
	}

	if !hasDir {
		t.Error("expected at least one directory (tree) entry")
	}
	if !hasFile {
		t.Error("expected at least one file (blob) entry")
	}
}

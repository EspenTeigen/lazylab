package views

import (
	"testing"

	"github.com/EspenTeigen/lazylab/internal/gitlab"
)

func TestGroupItem(t *testing.T) {
	group := gitlab.Group{
		ID:       1,
		Name:     "test-group",
		FullPath: "test-group",
	}
	item := GroupItem{Group: group}

	if item.Title() != "test-group" {
		t.Errorf("expected title 'test-group', got '%s'", item.Title())
	}

	if item.FilterValue() != "test-group" {
		t.Errorf("expected filter value 'test-group', got '%s'", item.FilterValue())
	}

	if item.Description() != "test-group" {
		t.Errorf("expected description 'test-group', got '%s'", item.Description())
	}
}

func TestNewGroups(t *testing.T) {
	groups := NewGroups()

	if groups.Title() != "Groups" {
		t.Errorf("expected title 'Groups', got '%s'", groups.Title())
	}

	if !groups.loading {
		t.Error("expected loading to be true initially")
	}
}

func TestProjectItem(t *testing.T) {
	project := gitlab.Project{
		ID:          101,
		Name:        "test-project",
		Description: "A test project",
		WebURL:      "https://gitlab.com/test/test-project",
	}
	item := ProjectItem{Project: project}

	if item.Title() != "test-project" {
		t.Errorf("expected title 'test-project', got '%s'", item.Title())
	}

	if item.Description() != "A test project" {
		t.Errorf("expected description 'A test project', got '%s'", item.Description())
	}
}

func TestNewProjects(t *testing.T) {
	projects := NewProjects(1, "test-group")

	if projects.Title() != "test-group" {
		t.Errorf("expected title 'test-group', got '%s'", projects.Title())
	}

	if projects.groupID != 1 {
		t.Errorf("expected groupID 1, got %d", projects.groupID)
	}
}

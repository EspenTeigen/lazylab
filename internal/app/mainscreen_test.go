package app

import (
	"strings"
	"testing"

	"github.com/EspenTeigen/lazylab/internal/gitlab"
)

func TestRebuildNavTree(t *testing.T) {
	m := &MainScreen{
		groups: []gitlab.Group{
			{ID: 1, Name: "Group A", FullPath: "group-a"},
			{ID: 2, Name: "Group B", FullPath: "group-b"},
		},
		expandedGroups: make(map[int]bool),
		groupProjects:  make(map[int][]gitlab.Project),
	}

	// Initial build - no groups expanded
	m.rebuildNavTree()

	if len(m.treeNodes) != 2 {
		t.Errorf("expected 2 nodes (collapsed groups), got %d", len(m.treeNodes))
	}

	if m.treeNodes[0].Type != "group" {
		t.Errorf("expected first node to be group, got '%s'", m.treeNodes[0].Type)
	}

	if m.treeNodes[0].Name != "Group A" {
		t.Errorf("expected 'Group A', got '%s'", m.treeNodes[0].Name)
	}
}

func TestRebuildNavTree_WithExpandedGroup(t *testing.T) {
	m := &MainScreen{
		groups: []gitlab.Group{
			{ID: 1, Name: "Group A", FullPath: "group-a"},
		},
		expandedGroups: map[int]bool{1: true},
		groupProjects: map[int][]gitlab.Project{
			1: {
				{ID: 10, Name: "Project 1", PathWithNamespace: "group-a/project-1"},
				{ID: 11, Name: "Project 2", PathWithNamespace: "group-a/project-2"},
			},
		},
	}

	m.rebuildNavTree()

	// Should have: 1 group + 2 projects = 3 nodes
	if len(m.treeNodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(m.treeNodes))
	}

	// First node: group
	if m.treeNodes[0].Type != "group" {
		t.Errorf("expected group, got '%s'", m.treeNodes[0].Type)
	}
	if !m.treeNodes[0].Expanded {
		t.Error("expected group to be expanded")
	}

	// Second node: project with depth 1
	if m.treeNodes[1].Type != "project" {
		t.Errorf("expected project, got '%s'", m.treeNodes[1].Type)
	}
	if m.treeNodes[1].Depth != 1 {
		t.Errorf("expected depth 1, got %d", m.treeNodes[1].Depth)
	}
}

func TestRebuildNavTree_CollapseGroup(t *testing.T) {
	m := &MainScreen{
		groups: []gitlab.Group{
			{ID: 1, Name: "Group A", FullPath: "group-a"},
		},
		expandedGroups: map[int]bool{1: true},
		groupProjects: map[int][]gitlab.Project{
			1: {
				{ID: 10, Name: "Project 1"},
			},
		},
	}

	// Build with expanded
	m.rebuildNavTree()
	if len(m.treeNodes) != 2 {
		t.Fatalf("expected 2 nodes when expanded, got %d", len(m.treeNodes))
	}

	// Collapse
	m.expandedGroups[1] = false
	m.rebuildNavTree()

	if len(m.treeNodes) != 1 {
		t.Errorf("expected 1 node when collapsed, got %d", len(m.treeNodes))
	}
}

func TestTreeNode_Properties(t *testing.T) {
	group := &gitlab.Group{ID: 1, Name: "Test Group", FullPath: "test-group"}
	node := TreeNode{
		Type:     "group",
		Name:     group.Name,
		FullPath: group.FullPath,
		ID:       group.ID,
		Depth:    0,
		Expanded: false,
		Group:    group,
	}

	if node.Type != "group" {
		t.Errorf("expected type 'group', got '%s'", node.Type)
	}
	if node.Group == nil {
		t.Error("expected Group pointer to be set")
	}
	if node.Project != nil {
		t.Error("expected Project pointer to be nil for group node")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"\x1b[31mred text\x1b[0m", "red text"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no escapes here", "no escapes here"},
		{"\x1b[38;5;196mcolor\x1b[0m normal", "color normal"},
	}

	for _, tt := range tests {
		result := stripANSI(tt.input)
		if result != tt.expected {
			t.Errorf("stripANSI(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected int // number of lines
	}{
		{"short", 10, 1},
		{"this is a longer line that should wrap", 10, 4},
		{"no wrap needed", 50, 1},
		{"", 10, 1},
	}

	for _, tt := range tests {
		result := wrapText(tt.input, tt.width)
		lines := strings.Split(result, "\n")
		if len(lines) < tt.expected {
			t.Errorf("wrapText(%q, %d) got %d lines, expected at least %d",
				tt.input, tt.width, len(lines), tt.expected)
		}
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	input := "some text"
	result := wrapText(input, 0)
	if result != input {
		t.Errorf("expected unchanged text for zero width, got '%s'", result)
	}
}

func TestRenderMarkdown(t *testing.T) {
	// Test empty content
	result := renderMarkdown("")
	if result != "" {
		t.Errorf("expected empty string for empty input, got '%s'", result)
	}

	// Test basic markdown
	result = renderMarkdown("# Hello")
	if result == "" {
		t.Error("expected non-empty result for markdown input")
	}
	// Glamour should render something
	if !strings.Contains(result, "Hello") {
		t.Error("expected rendered output to contain 'Hello'")
	}
}

func TestHighlightCode(t *testing.T) {
	code := `package main

func main() {
    println("Hello")
}`

	result := highlightCode(code, "main.go")
	if result == "" {
		t.Error("expected non-empty highlighted output")
	}

	// Should contain the original code content
	if !strings.Contains(result, "main") {
		t.Error("expected output to contain 'main'")
	}
}

func TestHighlightCode_UnknownLanguage(t *testing.T) {
	code := "some random content"
	result := highlightCode(code, "unknown.xyz")

	// Should still return something (fallback)
	if result == "" {
		t.Error("expected non-empty output even for unknown file type")
	}
}

func TestNewMainScreen(t *testing.T) {
	// This will try to load credentials, which may fail in test environment
	// but the screen should still be created
	screen := NewMainScreen()

	if screen == nil {
		t.Fatal("expected non-nil MainScreen")
	}

	if screen.focusedPanel != PanelNavigator {
		t.Errorf("expected initial focus on PanelNavigator, got %d", screen.focusedPanel)
	}

	if screen.expandedGroups == nil {
		t.Error("expected expandedGroups map to be initialized")
	}

	if screen.groupProjects == nil {
		t.Error("expected groupProjects map to be initialized")
	}
}

func TestContentTabNames(t *testing.T) {
	if len(contentTabNames) != int(TabCount) {
		t.Errorf("expected %d tab names, got %d", TabCount, len(contentTabNames))
	}

	expectedNames := []string{"Files", "MRs", "Pipelines"}
	for i, name := range expectedNames {
		if contentTabNames[i] != name {
			t.Errorf("expected tab %d to be '%s', got '%s'", i, name, contentTabNames[i])
		}
	}
}

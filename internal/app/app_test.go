package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// mockView implements views.View for testing
type mockView struct {
	title string
}

func (m mockView) Init() tea.Cmd                        { return nil }
func (m mockView) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m mockView) View() string                        { return m.title }
func (m mockView) Title() string                       { return m.title }
func (m mockView) ShortHelp() []key.Binding            { return nil }

func TestNewApp(t *testing.T) {
	view := mockView{title: "Test View"}
	app := New(view)

	if app.stack.Len() != 1 {
		t.Errorf("expected 1 view in stack, got %d", app.stack.Len())
	}

	if app.stack.Current().Title() != "Test View" {
		t.Errorf("expected title 'Test View', got '%s'", app.stack.Current().Title())
	}
}

func TestViewStack(t *testing.T) {
	stack := NewViewStack()

	if stack.Len() != 0 {
		t.Errorf("expected empty stack, got %d", stack.Len())
	}

	view1 := mockView{title: "View 1"}
	view2 := mockView{title: "View 2"}

	stack.Push(view1)
	stack.Push(view2)

	if stack.Len() != 2 {
		t.Errorf("expected 2 views, got %d", stack.Len())
	}

	if stack.Current().Title() != "View 2" {
		t.Errorf("expected current 'View 2', got '%s'", stack.Current().Title())
	}

	popped := stack.Pop()
	if popped.Title() != "View 2" {
		t.Errorf("expected popped 'View 2', got '%s'", popped.Title())
	}

	if stack.Len() != 1 {
		t.Errorf("expected 1 view after pop, got %d", stack.Len())
	}
}

func TestBreadcrumbs(t *testing.T) {
	stack := NewViewStack()
	stack.Push(mockView{title: "Groups"})
	stack.Push(mockView{title: "Projects"})
	stack.Push(mockView{title: "my-project"})

	breadcrumbs := stack.Breadcrumbs()

	expected := []string{"Groups", "Projects", "my-project"}
	if len(breadcrumbs) != len(expected) {
		t.Errorf("expected %d breadcrumbs, got %d", len(expected), len(breadcrumbs))
	}

	for i, b := range breadcrumbs {
		if b != expected[i] {
			t.Errorf("expected breadcrumb[%d] '%s', got '%s'", i, expected[i], b)
		}
	}
}

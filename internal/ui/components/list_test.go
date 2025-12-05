package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestNewItem(t *testing.T) {
	item := NewItem("123", "Test Title", "Test Description")

	if item.ID() != "123" {
		t.Errorf("expected ID '123', got '%s'", item.ID())
	}
	if item.Title() != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", item.Title())
	}
	if item.Description() != "Test Description" {
		t.Errorf("expected description 'Test Description', got '%s'", item.Description())
	}
	if item.FilterValue() != "Test Title" {
		t.Errorf("expected filter value 'Test Title', got '%s'", item.FilterValue())
	}
}

func TestNewList(t *testing.T) {
	items := []list.Item{
		NewItem("1", "Item 1", "Desc 1"),
		NewItem("2", "Item 2", "Desc 2"),
	}

	l := NewList("Test List", items, 80, 24)

	if len(l.Items()) != 2 {
		t.Errorf("expected 2 items, got %d", len(l.Items()))
	}
}

func TestListSelectedItem(t *testing.T) {
	items := []list.Item{
		NewItem("1", "Item 1", "Desc 1"),
		NewItem("2", "Item 2", "Desc 2"),
	}

	l := NewList("Test List", items, 80, 24)

	selected := l.SelectedItem()
	if selected == nil {
		t.Error("expected selected item, got nil")
		return
	}

	item, ok := selected.(Item)
	if !ok {
		t.Error("expected Item type")
		return
	}

	if item.ID() != "1" {
		t.Errorf("expected first item selected, got ID '%s'", item.ID())
	}
}

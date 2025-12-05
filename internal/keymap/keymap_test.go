package keymap

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"Up", km.Up, []string{"k", "up"}},
		{"Down", km.Down, []string{"j", "down"}},
		{"Select", km.Select, []string{"enter"}},
		{"Quit", km.Quit, []string{"q", "ctrl+c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindingKeys := tt.binding.Keys()
			if len(bindingKeys) != len(tt.keys) {
				t.Errorf("expected %d keys, got %d", len(tt.keys), len(bindingKeys))
			}
			for i, k := range tt.keys {
				if bindingKeys[i] != k {
					t.Errorf("expected key %s, got %s", k, bindingKeys[i])
				}
			}
		})
	}
}

func TestShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.ShortHelp()

	if len(help) != 6 {
		t.Errorf("expected 6 short help bindings, got %d", len(help))
	}
}

func TestFullHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.FullHelp()

	if len(help) != 4 {
		t.Errorf("expected 4 help columns, got %d", len(help))
	}
}

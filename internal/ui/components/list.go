package components

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/keymap"
	"github.com/espen/lazylab/internal/ui/styles"
)

// Item represents a list item with ID and display text
type Item struct {
	id          string
	title       string
	description string
}

func (i Item) ID() string          { return i.id }
func (i Item) Title() string       { return i.title }
func (i Item) Description() string { return i.description }
func (i Item) FilterValue() string { return i.title }

// NewItem creates a new list item
func NewItem(id, title, description string) Item {
	return Item{id: id, title: title, description: description}
}

// List wraps bubbles/list with our styling and keybindings
type List struct {
	list   list.Model
	keymap keymap.KeyMap
	width  int
	height int
}

// NewList creates a new styled list
func NewList(title string, items []list.Item, width, height int) List {
	delegate := list.NewDefaultDelegate()

	// Style the delegate
	delegate.Styles.SelectedTitle = styles.SelectedItem
	delegate.Styles.SelectedDesc = styles.SelectedItem.Foreground(styles.ColorGray)
	delegate.Styles.NormalTitle = styles.NormalItem
	delegate.Styles.NormalDesc = styles.DimmedText

	l := list.New(items, delegate, width, height)
	l.Title = title
	l.Styles.Title = styles.ActivePanelTitle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return List{
		list:   l,
		keymap: keymap.DefaultKeyMap(),
		width:  width,
		height: height,
	}
}

// Init initializes the list
func (l List) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (l List) Update(msg tea.Msg) (List, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		l.list.SetSize(msg.Width, msg.Height-4)
	case tea.KeyMsg:
		if key.Matches(msg, l.keymap.Top) {
			l.list.Select(0)
			return l, nil
		}
		if key.Matches(msg, l.keymap.Bottom) {
			l.list.Select(len(l.list.Items()) - 1)
			return l, nil
		}
	}

	var cmd tea.Cmd
	l.list, cmd = l.list.Update(msg)
	return l, cmd
}

// View renders the list
func (l List) View() string {
	return l.list.View()
}

// SelectedItem returns the currently selected item
func (l List) SelectedItem() list.Item {
	return l.list.SelectedItem()
}

// SetItems updates the list items
func (l *List) SetItems(items []list.Item) tea.Cmd {
	return l.list.SetItems(items)
}

// Items returns all items
func (l List) Items() []list.Item {
	return l.list.Items()
}

// Index returns the currently selected index
func (l List) Index() int {
	return l.list.Index()
}

// SetSize updates the list dimensions
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.list.SetSize(width, height)
}

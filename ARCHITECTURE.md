# GitLab TUI Architecture

A lazygit-style terminal UI for GitLab.

## Tech Stack

- **Go 1.22+**
- **bubbletea** - TUI framework (Elm architecture)
- **lipgloss** - Styling (borders, colors, layout)
- **bubbles** - Components (list, table, viewport, textinput)
- **gitlab.com/gitlab-org/api/client-go** - Official GitLab API client

## Project Structure

```
gitlab-tui/
├── cmd/
│   └── gitlab-tui/
│       └── main.go              # Entry point
├── internal/
│   ├── app/
│   │   └── app.go               # Root bubbletea model, orchestrates views
│   ├── ui/
│   │   ├── views/
│   │   │   ├── groups.go        # Group navigator
│   │   │   ├── projects.go      # Project list
│   │   │   ├── project.go       # Single project view (tabs: MRs, pipelines, branches)
│   │   │   ├── mrs.go           # Merge requests list
│   │   │   ├── mr.go            # Single MR detail
│   │   │   ├── pipelines.go     # Pipeline list
│   │   │   ├── pipeline.go      # Pipeline detail (jobs)
│   │   │   ├── branches.go      # Branch list
│   │   │   └── commits.go       # Commit list
│   │   ├── components/
│   │   │   ├── list.go          # Reusable list wrapper
│   │   │   ├── tabs.go          # Tab bar component
│   │   │   ├── statusbar.go     # Bottom status bar
│   │   │   ├── help.go          # Help overlay
│   │   │   └── spinner.go       # Loading indicator
│   │   └── styles/
│   │       └── styles.go        # Lipgloss style definitions
│   ├── gitlab/
│   │   ├── client.go            # API client wrapper
│   │   ├── groups.go            # Group operations
│   │   ├── projects.go          # Project operations
│   │   ├── mrs.go               # MR operations
│   │   ├── pipelines.go         # Pipeline operations
│   │   └── branches.go          # Branch/commit operations
│   ├── config/
│   │   ├── config.go            # Config loading (XDG paths)
│   │   └── auth.go              # Token management (steal from glab)
│   └── keymap/
│       └── keymap.go            # Keybinding definitions
├── go.mod
├── go.sum
├── ARCHITECTURE.md
└── README.md
```

## Navigation Model

```
┌─────────────────────────────────────────────────────────────┐
│  Groups  →  Projects  →  Project Detail  →  Item Detail     │
│                              │                              │
│                         ┌────┴────┐                         │
│                         │  Tabs   │                         │
│                         ├─────────┤                         │
│                         │ MRs     │                         │
│                         │ Pipes   │                         │
│                         │ Branches│                         │
│                         │ Commits │                         │
│                         └─────────┘                         │
└─────────────────────────────────────────────────────────────┘
```

**View Stack:**
- Push views onto stack when drilling down
- Pop with `Esc` or `q` to go back
- `?` toggles help overlay at any level

## Keybindings (vim-style)

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `h/l` | Navigate tabs (in project view) |
| `Enter` | Select / drill down |
| `Esc` | Go back |
| `q` | Quit (or go back if not root) |
| `g/G` | Jump to top/bottom |
| `Ctrl+d/u` | Half-page down/up |
| `/` | Search/filter |
| `r` | Refresh |
| `?` | Toggle help |
| `n` | New (context-dependent: project, MR, etc.) |
| `o` | Open in browser |

## State Management

Each view is a bubbletea `Model` with:
- `Init()` - Fetch initial data
- `Update(msg)` - Handle input, API responses
- `View()` - Render

Root `App` model manages:
- View stack (navigation history)
- Global state (current group/project context)
- Auth token

## API Client Pattern

```go
type Client struct {
    gl *gitlab.Client
}

// All methods return channels for async operation
func (c *Client) ListGroups(ctx context.Context) tea.Cmd {
    return func() tea.Msg {
        groups, _, err := c.gl.Groups.ListGroups(...)
        if err != nil {
            return ErrMsg{err}
        }
        return GroupsMsg{groups}
    }
}
```

## Config & Auth

Follow XDG spec (like glab):
- `~/.config/gitlab-tui/config.yaml` - Settings
- Token from env `GITLAB_TOKEN` or config file
- Support multiple instances (gitlab.com + self-hosted)

```yaml
# ~/.config/gitlab-tui/config.yaml
default_host: gitlab.com
hosts:
  gitlab.com:
    token: ${GITLAB_TOKEN}  # or hardcoded
  gitlab.company.com:
    token: ${COMPANY_GITLAB_TOKEN}
```

## MVP Scope (v0.1)

1. Auth with personal access token
2. Browse groups (nested)
3. Browse projects in group
4. View project MRs
5. View project pipelines
6. View branches
7. View commits on branch

## Future (post-MVP)

- Create project
- Create MR
- Retry/cancel pipelines
- Fuzzy search
- Multiple instance switching
- OAuth flow

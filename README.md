# LazyLab — A Terminal UI for GitLab

Browse GitLab projects, merge requests, pipelines, and streaming job logs from your terminal. A fast, keyboard-driven TUI client for GitLab and self-hosted GitLab instances, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

[![Go Report Card](https://goreportcard.com/badge/github.com/EspenTeigen/lazylab)](https://goreportcard.com/report/github.com/EspenTeigen/lazylab)
[![Release](https://img.shields.io/github/v/release/EspenTeigen/lazylab)](https://github.com/EspenTeigen/lazylab/releases)
[![License](https://img.shields.io/github/license/EspenTeigen/lazylab)](LICENSE)

## Features

- Browse groups and projects in a tree view
- View repository files with syntax highlighting
- View merge requests with status indicators (open, merged, closed, draft)
- View pipelines with stage status aggregation
- **Live-streaming pipeline job logs** with auto-refresh and horizontal scrolling
- **Global running/pending jobs popup** across all projects
- Auto-refreshing pipeline status
- **Browse releases** and download assets
- Switch branches
- Rendered README preview (markdown via Glamour)
- **Vim-style visual line selection** — select and yank lines from files, README, and logs
- **Copy clone URLs** (SSH and HTTPS) to clipboard
- Panel-based layout with focus indicators
- Cross-platform clipboard support (macOS, Linux Wayland/X11, Windows)
- Works with GitLab.com and self-hosted instances
- Demo mode (`--demo`) for testing without API access

## Installation

### Quick install (recommended)

```bash
curl -sL https://raw.githubusercontent.com/EspenTeigen/lazylab/main/install.sh | bash
```

Or with a custom install directory:

```bash
INSTALL_DIR=/usr/local/bin curl -sL https://raw.githubusercontent.com/EspenTeigen/lazylab/main/install.sh | bash
```

### From releases

Download the latest binary from the [releases page](https://github.com/EspenTeigen/lazylab/releases).

### Using go install

```bash
go install github.com/EspenTeigen/lazylab/cmd/lazylab@latest
```

### Build and install locally

```bash
git clone https://github.com/EspenTeigen/lazylab.git
cd lazylab
make install
```

This installs to `~/.local/bin` by default. Override with:

```bash
make install INSTALL_DIR=/usr/local/bin
```

To uninstall:

```bash
make uninstall
```

## Authentication

On first run, lazylab will prompt for your GitLab host and token.

### Reconfigure / Fix login issues

If you need to change your GitLab instance, update your token, or fix a failed login:

```bash
lazylab --setup
```

This forces the setup screen to appear, allowing you to enter new credentials.

### Environment variables

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
export GITLAB_HOST="gitlab.mycompany.com"  # optional, defaults to gitlab.com
```

### Config file

Create `~/.config/lazylab/config.yaml`:

```yaml
default_host: gitlab.com
hosts:
  gitlab.com:
    token: glpat-xxxxxxxxxxxx
  gitlab.mycompany.com:
    token: glpat-yyyyyyyyyyyy
```

### glab CLI

If you use [glab](https://gitlab.com/gitlab-org/cli), lazylab will automatically use its stored credentials.

## Keybindings

### General

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `h/l` | Switch tabs |
| `Enter` | Select / expand |
| `Esc` | Go back / close popup |
| `g/G` | Go to top/bottom |
| `C-d/C-u` | Page down/up |
| `o` | Open in browser |
| `r` | Refresh / retry on error |
| `R` | Show all running/pending jobs |
| `S` | Copy SSH clone URL |
| `U` | Copy HTTPS clone URL |
| `q` | Quit |

### Panel focus

| Key | Action |
|-----|--------|
| `H` / `Shift+Left` / `1` | Focus navigator panel |
| `L` / `Shift+Right` / `2` | Focus content panel |
| `K` / `Shift+Up` / `3` | Focus README panel |

### Visual mode (files, README, logs)

| Key | Action |
|-----|--------|
| `V` | Toggle visual line selection |
| `yy` | Yank current line |
| `ggy` | Yank all content |

### Files tab

| Key | Action |
|-----|--------|
| `b` | Switch branch |

### Pipeline job log popup

| Key | Action |
|-----|--------|
| `j/k` | Switch between jobs |
| `C-d/C-u` | Scroll log |
| `g/G` | Go to top/bottom of log |
| `0/$` | Scroll to start/end of line |
| `y` | Copy log to clipboard |
| `V` | Visual line selection |
| `Esc` | Close |

### Releases tab

| Key | Action |
|-----|--------|
| `y` / `Enter` | Copy asset URL |
| `o` | Copy release URL |
| `d` | Download asset |
| `~` | Jump to home directory (in folder browser) |

## Security

**This application is strictly read-only.** It will never modify any data on your GitLab instance.

- All write operations (POST, PUT, PATCH, DELETE) are blocked at the client level
- Only `read_api` scope is required - no write permissions needed
- Safety checks are enforced in code and covered by tests

You can safely use lazylab with your production GitLab instance without worrying about accidental modifications.

## Requirements

- GitLab Personal Access Token with `read_api` scope
- Terminal with true color support (recommended)
- **Linux clipboard support** (optional):
  ```bash
  # Wayland (Sway, GNOME, etc.)
  sudo pacman -S wl-clipboard  # Arch
  sudo apt install wl-clipboard  # Debian/Ubuntu

  # X11
  sudo pacman -S xclip  # Arch
  sudo apt install xclip  # Debian/Ubuntu
  ```

## Contributing

Pull requests are welcome. However, I'm not accepting feature requests - this is a personal project and I'll add features as I need them. If you want a feature, feel free to fork and implement it yourself, or submit a PR.

## Acknowledgments

This project is built with these excellent open source libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Chroma](https://github.com/alecthomas/chroma) - Syntax highlighting

Inspired by [lazygit](https://github.com/jesseduffield/lazygit).

## Disclaimer

This software is provided "as is", without warranty of any kind. Use at your own risk.

## License

MIT

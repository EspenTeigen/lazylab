# lazylab

A terminal UI for GitLab, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

> **Note:** This is a vibe-coded project. I haven't written a single line of code myself - it's entirely AI-generated using [Claude Code](https://claude.ai/claude-code). I built this because I wanted it, not as a showcase of coding skills. If the code style isn't perfect, I don't care - it works for my use case.

![lazylab demo](https://via.placeholder.com/800x400?text=lazylab+screenshot)

## Features

- Browse groups and projects in a tree view
- View repository files with syntax highlighting
- View merge requests and pipelines
- Switch branches
- View pipeline job logs
- Rendered README preview (markdown)
- Works with GitLab.com and self-hosted instances

## Installation

### Quick install (recommended)

```bash
curl -sL https://raw.githubusercontent.com/espen/lazylab/main/install.sh | bash
```

Or with a custom install directory:

```bash
INSTALL_DIR=/usr/local/bin curl -sL https://raw.githubusercontent.com/espen/lazylab/main/install.sh | bash
```

### From releases

Download the latest binary from the [releases page](https://github.com/espen/lazylab/releases).

### Using go install

```bash
go install github.com/espen/lazylab/cmd/lazylab@latest
```

### Build and install locally

```bash
git clone https://github.com/espen/lazylab.git
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

On first run, lazylab will prompt for your GitLab token. You can also configure it via:

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

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `h/l` | Navigate left/right, switch tabs |
| `Enter` | Select / expand |
| `Esc` | Go back / close popup |
| `b` | Switch branch (in files view) |
| `1-4` | Focus panel |
| `q` | Quit |

### Pipeline job log popup

| Key | Action |
|-----|--------|
| `j/k` | Switch between jobs |
| `h/l` | Scroll log |
| `y` | Copy log to clipboard |
| `Esc` | Close |

## Requirements

- GitLab Personal Access Token with `read_api` scope
- Terminal with true color support (recommended)
- **Linux only:** `xclip` or `xsel` for clipboard support (optional)
  ```bash
  # Debian/Ubuntu
  sudo apt install xclip
  # Fedora
  sudo dnf install xclip
  # Arch
  sudo pacman -S xclip
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

## License

MIT

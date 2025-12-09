# lazylab

A terminal UI for GitLab, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

> **Note:** This is a vibe-coded project. I haven't written a single line of code myself - it's entirely AI-generated using [Claude Code](https://claude.ai/claude-code). I built this because I wanted it, not as a showcase of coding skills. If the code style isn't perfect, I don't care - it works for my use case.

## Features

- Browse groups and projects in a tree view
- View repository files
- View merge requests and pipelines
- **Live-streaming pipeline job logs** with auto-refresh
- Auto-refreshing pipeline status
- Switch branches
- Rendered README preview (markdown)
- Works with GitLab.com and self-hosted instances

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

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `h/l` | Switch tabs |
| `Enter` | Select / expand |
| `Esc` | Go back / close popup |
| `g/G` | Go to top/bottom |
| `C-d/C-u` | Page down/up |
| `b` | Switch branch (in files view) |
| `o` | Open in browser |
| `r` | Refresh / retry on error |
| `q` | Quit |

### Pipeline job log popup

| Key | Action |
|-----|--------|
| `j/k` | Switch between jobs |
| `C-d/C-u` | Scroll log |
| `g/G` | Go to top/bottom of log |
| `y` | Copy log to clipboard |
| `Esc` | Close |

## Security

**This application is strictly read-only.** It will never modify any data on your GitLab instance.

- All write operations (POST, PUT, PATCH, DELETE) are blocked at the client level
- Only `read_api` scope is required - no write permissions needed
- Safety checks are enforced in code and covered by tests

You can safely use lazylab with your production GitLab instance without worrying about accidental modifications.

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

## Disclaimer

This software is provided "as is", without warranty of any kind. Use at your own risk.

## License

MIT

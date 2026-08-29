# git-remote-color

> A **colorized, GitHub-aware replacement for `git remote -v`** with rich metadata, GitHub Actions workflow status, README preview, smart caching, offline support, and repo lookup by name.

[![Release](https://img.shields.io/github/v/release/cumulus13/git-remote-color?color=blue)](https://github.com/cumulus13/git-remote-color/releases)
[![Downloads](https://img.shields.io/github/downloads/cumulus13/git-remote-color/total)](https://github.com/cumulus13/git-remote-color/releases)
[![License](https://img.shields.io/github/license/cumulus13/git-remote-color?color=green)](https://github.com/cumulus13/git-remote-color/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Build](https://img.shields.io/github/actions/workflow/status/cumulus13/git-remote-color/release.yml?label=build)](https://github.com/cumulus13/git-remote-color/actions)

---

## ✨ Features

### 🎨 Rich Colored Output
- Truecolor ANSI (24-bit) with short `#RGB` support
- Fully configurable via JSON
- Clean, structured CLI layout

### ⚙️ GitHub Actions Workflow Status (NEW)
- View the latest workflow run per workflow with `-w` / `--workflows`
- Status icons: ✔ success, ✘ failure, ⟳ in_progress, ⊘ cancelled, ⏱ timed_out, and more
- Shows workflow name, branch, event trigger, and relative time ("3h ago")
- Compact inline CI summary even without `-w` (shows latest run in main view)

### 🔎 Repo Lookup by Name (NEW)
- Pass `owner/repo` or just `repo` (with `owner` set in config) to fetch info without a local clone
- Example: `git-remote-color myrepo` → resolves to `yourname/myrepo` via `"owner"` in config

### 📖 README Preview
- Fetch and display remote README with `-d` / `--detail` / `-r` / `--readme`
- **Glow-like rendering** powered by [Glamour](https://github.com/charmbracelet/glamour)
- Syntax highlighting, styled tables, and proper markdown formatting
- Smart pager integration; use `-f` to bypass

| Content Length | Default Behavior | With `-f` Flag |
|---------------|------------------|----------------|
| ≤ 50 lines    | Direct output    | Direct output  |
| > 50 lines    | Opens in pager   | Direct output  |

### 🌐 GitHub Deep Info
- 📝 Description
- 🌍 Public / 🔒 Private
- ⭐ Stars / 🍴 Forks / 🐞 Issues / ⬇ Downloads
- 🧠 Languages with percentage (sorted, color-coded)
- 🌿 Branch list (with ★ default branch marker)
- 🏷️ Tag list
- ⚙️ Workflow status

### ⚡ Smart Cache
- In-memory cache (1 hour TTL) with RW-mutex for goroutine safety
- `--no-cache` to force a fresh fetch
- Graceful offline fallback to stale cache

### 🔗 Git Integration
- Works from any subdirectory (walks up to Git root)
- Supports: `.`, `../path`, `/absolute/path`, `~/path`, `owner/repo`, `reponame`
- Parallel API calls (releases, branches, tags, languages, workflows fetched concurrently)

---

## 🚀 Usage

```bash
# Current directory
git-remote-color

# With README (pager for long content)
git-remote-color -d

# With workflow status
git-remote-color -w

# Full combo: README + workflows, no pager
git-remote-color -dfw

# Lookup a repo directly (owner set in config)
git-remote-color myrepo

# Lookup with explicit owner/repo
git-remote-color cumulus13/git-remote-color

# Specific path
git-remote-color /path/to/repo -d

# Pipe to file
git-remote-color -dfw > report.txt

# Bypass cache
git-remote-color --no-cache -w

# Version
git-remote-color -v
```

---

## 🎯 Example Output

### Basic Output
```
origin  https://github.com/user/repo  (fetch, push)
   A powerful CLI tool
   🌍 public  ⭐ 42  🍴 10  🐞 3  ⬇ 2500  🕒 2024-01-15
   🧠 Go 70.0%  Shell 20.0%  Makefile 10.0%
   ⚙ CI: ✔ success  Release [main]

   🌿 branches:
     - main ★
     - dev

   🏷️  tags:
     - v1.0
     - v1.1
```

### With Workflows (-w)
```
═══ GitHub Actions ═══
   📋 3 total run(s), showing last 3

   ✔ success   Release [main] (push)  2h ago
   ✔ success   CI [dev] (push)  1d ago
   ⊘ cancelled  Nightly [main] (schedule)  3d ago
```

---

## ⚙️ Configuration

Place a `gitv.json` (or `git-remote-color.json`) in one of the auto-detected locations.

```json
{
  "remote": "#00FFFF",
  "scheme": "#FFAAFF",
  "host": "#55AA00",
  "path": "#AAAAFF",
  "repo": "#FFFF00",
  "fetch": "#00AAFF",
  "push": "#AA5500",
  "description": "#00AAFF",
  "branch": "#FFAAFF",
  "tag": "#AAAA00",
  "visibility": "#00FFFF",
  "last_update": "#FFFF00",
  "readme_color": "#95E1D3",
  "workflow_color": "#C3E88D",
  "github_token": "",
  "owner": "yourname",
  "tokens": {
    "yourname": "ghp_xxx_personal_account_token",
    "your-org": "ghp_yyy_org_or_second_account_token"
  },
  "glamour_style": "auto",
  "glamour_width": 100,
  "language_colors": [
    "#FF5555", "#55FF55", "#5599FF",
    "#FFFF55", "#FF55FF", "#55FFFF", "#FFA500"
  ]
}
```

### New Fields in v1.1

| Field            | Description                                      | Default        |
|------------------|--------------------------------------------------|----------------|
| `workflow_color` | Color for workflow names                         | `"#C3E88D"`    |
| `owner`          | Default GitHub owner for bare repo name lookups  | `""`           |
| `tokens`         | Per-owner GitHub tokens, `{ "owner": "token" }`  | `{}`           |

Config lookup order:
1. `$GIT_REMOTE_COLOR_CONFIG` env var
2. Executable directory
3. Current working directory
4. Platform config dir (`%AppData%` / `~/.config` / `~/Library/Application Support`)
5. Home directory (dotfile variants)

### GitHub Token
Set `github_token` in config, or export `GITHUB_TOKEN` in your shell.

| Case         | Behavior              |
|-------------|----------------------|
| Public repo  | Works without token   |
| Private repo | Requires token        |

### Multiple Accounts / Tokens

If your remotes span more than one GitHub account or org (e.g. a personal
account and a work org), a single `github_token` can't authenticate as both.
Add a `tokens` map keyed by owner (case-insensitive) — each remote is then
authenticated with the token matching its own owner:

```json
"tokens": {
  "cumulus13": "ghp_personal_token",
  "licface": "ghp_other_account_token",
  "my-work-org": "ghp_org_token"
}
```

Resolution order per repo owner:
1. `tokens[owner]` in config (case-insensitive)
2. `GITHUB_TOKEN_<OWNER>` env var (e.g. `GITHUB_TOKEN_LICFACE`, non-alphanumeric
   characters become `_`)
3. `github_token` in config, or `GITHUB_TOKEN` env var (global fallback)

This applies everywhere a token is used: repo metadata, README, and workflow
status — so `licface/nettop2` and `cumulus13/nettop2` remotes in the same
working copy each authenticate correctly even though they belong to different
accounts.
| Rate limit   | Token increases to 5000 req/hr |

---

## 📦 Installation

### From Source
```bash
go install github.com/cumulus13/git-remote-color@latest
```

### From Releases
Download the latest binary from [Releases](https://github.com/cumulus13/git-remote-color/releases).

| Platform | Binary Name |
|----------|-------------|
| Linux x64 | `git-remote-color_*_linux_amd64` |
| Linux ARM64 | `git-remote-color_*_linux_arm64` |
| Linux ARMv7 (RPi 3/4, Termux) | `git-remote-color_*_linux_armv7` |
| Linux ARMv6 (RPi Zero/1) | `git-remote-color_*_linux_armv6` |
| Android ARM64 (Termux) | `git-remote-color_*_android_arm64_termux` |
| Android ARMv7 (Termux) | `git-remote-color_*_android_armv7_termux` |
| macOS Intel | `git-remote-color_*_darwin_amd64` |
| macOS Apple Silicon | `git-remote-color_*_darwin_arm64` |
| Windows x64 | `git-remote-color_*_windows_amd64.exe` |
| Windows ARM64 | `git-remote-color_*_windows_arm64.exe` |
| FreeBSD x64 | `git-remote-color_*_freebsd_amd64` |

### Termux (Android)
```bash
# ARM64 device (most modern Android phones)
curl -L https://github.com/cumulus13/git-remote-color/releases/latest/download/git-remote-color_latest_android_arm64_termux \
  -o $PREFIX/bin/git-remote-color
chmod +x $PREFIX/bin/git-remote-color

# ARMv7 device
curl -L https://github.com/cumulus13/git-remote-color/releases/latest/download/git-remote-color_latest_android_armv7_termux \
  -o $PREFIX/bin/git-remote-color
chmod +x $PREFIX/bin/git-remote-color
```

### Homebrew (macOS/Linux)
```bash
brew install cumulus13/tap/git-remote-color
```

### Scoop (Windows)
```powershell
scoop bucket add cumulus13 https://github.com/cumulus13/scoop-bucket
scoop install git-remote-color
```

---

## 🔧 Flags Reference

| Flag | Aliases | Description |
|------|---------|-------------|
| `-d` | `--detail`, `-r`, `--readme` | Show README |
| `-w` | `--workflows` | Show GitHub Actions workflow runs |
| `-f` | `--full` | Disable pager, print directly |
| `--no-cache` | | Bypass in-memory cache |
| `-v` | `--version` | Show version |
| `-h` | `--help` | Show help |

Flags can be combined: `-dw`, `-df`, `-dfw`.

---

## 🧠 Behavior Summary

| Scenario              | Result                          |
|----------------------|---------------------------------|
| Same fetch/push       | Grouped as one remote           |
| Different fetch/push  | Both shown                      |
| Multiple remotes      | All shown                       |
| Subdirectory          | Walks up to Git root            |
| Bare name `myrepo`    | Resolved via config `owner`     |
| `owner/repo` slug     | Direct GitHub lookup (no clone) |
| Offline + cache       | Stale cache shown               |
| Offline, no cache     | Warning shown                   |
| Rate limited          | Informative error + hint        |
| README > 50 lines     | Opens in pager                  |
| README + `-f`         | Direct output                   |
| Workflow runs         | Deduplicated per workflow name  |
| Non-GitHub remote     | Shown colored, no metadata      |
| Windows               | Uses `more.com` as pager        |

---

## 🗺️ Roadmap

- [ ] GitLab and Bitbucket API support
- [ ] Config generator command (`git-remote-color --init-config`)
- [ ] Shell completion scripts (bash, zsh, fish)
- [ ] Multiple output formats (JSON, YAML)
- [ ] Watch mode for live updates
- [ ] Pull request summary

---

## 📄 License

MIT

---

## 👤 Author

[Hadi Cahyadi](mailto:cumulus13@gmail.com)

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)

[Support me on Patreon](https://www.patreon.com/cumulus13)

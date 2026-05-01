# git-remote-color

> A **colorized, GitHub-aware replacement for `git remote -v`** with rich metadata, README preview with Glow-like rendering, smart caching, and offline support.

[![Release](https://img.shields.io/github/v/release/cumulus13/git-remote-color?color=blue)](https://github.com/cumulus13/git-remote-color/releases)
[![Downloads](https://img.shields.io/github/downloads/cumulus13/git-remote-color/total)](https://github.com/cumulus13/git-remote-color/releases)
[![License](https://img.shields.io/github/license/cumulus13/git-remote-color?color=green)](https://github.com/cumulus13/git-remote-color/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.20+-00ADD8?logo=go)](https://go.dev/)
[![Build](https://img.shields.io/github/actions/workflow/status/cumulus13/git-remote-color/release.yml?label=build)](https://github.com/cumulus13/git-remote-color/actions)

---

## ✨ Features

### 🎨 Rich Colored Output

* Truecolor ANSI (24-bit)
* Fully configurable via JSON
* Clean, structured CLI layout

---

### 📖 README Preview (NEW)

* Fetch and display remote README with `-d` / `--detail` / `-r` / `--readme`
* **Glow-like rendering** powered by [Glamour](https://github.com/charmbracelet/glamour)
* Syntax highlighting, styled tables, and proper markdown formatting
* Smart pager integration for long content
* Control output with `-f` / `--full` flag

#### Pager Behavior:
| Content Length | Default Behavior | With `-f` Flag |
|---------------|------------------|----------------|
| ≤ 50 lines    | Direct output    | Direct output  |
| > 50 lines    | Opens in pager   | Direct output  |

---

### 🔗 Git Smart Integration

* Works like `git remote -v`
* Supports:
  * current directory
  * subdirectories inside repo
  * relative path
  * absolute path
  * home directory (`~`)
* Automatically detects Git root

---

### 🌐 GitHub Deep Info

For GitHub repositories:

* 📝 Description
* 🌍 Public / 🔒 Private
* ⭐ Stars / 🍴 Forks / 🐞 Issues / ⬇ Downloads
* 🧠 **Languages with percentage (sorted, color-coded)**
* 🌿 Branch list (with ★ default branch marker)
* 🏷️ Tag list
* 📄 README preview (optional)

---

### 🧠 Language Breakdown (Improved)

Instead of a single language:

```text
🧠 JavaScript 82.4%, HTML 12.1%, CSS 5.5%
```

* Sorted by usage (descending)
* Based on GitHub language API
* Accurate percentage calculation
* Each language gets a unique color from configurable palette

---

### ⚡ Smart Cache System

* Automatic in-memory cache (1 hour TTL)
* Prevents duplicate API calls
* Faster repeated runs

#### Cache behavior:

| Scenario  | Behavior               |
| --------- | ---------------------- |
| First run | Fetch from GitHub      |
| Next runs | Use cache (shows cached indicator) |
| Offline   | Use cache if available |

---

### 📡 Offline Support

When internet is unavailable:

#### ✔ With cache:

```text
(cached)
🌍 public ⭐ 10 🍴 2 🐞 1 ⬇ 1500 🕒 2024-01-15
```

#### ❌ Without cache:

```text
⚠ offline (no cached data)
```

👉 No fake data is shown.

---

### 🧠 Smart Token Handling

Optional GitHub token:

```json
{
  "github_token": "ghp_xxxxx"
}
```

Behavior:

| Case         | Behavior              |
| ------------ | --------------------- |
| Public repo  | Works without token   |
| Private repo | Requires token        |
| Rate limit   | Token increases limit |

---

## 🚀 Usage

### Basic Usage
```bash
# Current directory
git-remote-color

# Relative path
git-remote-color ../project

# Absolute path
git-remote-color /home/user/projects/repo

# Home directory
git-remote-color ~/code/myproject
```

### README Preview
```bash
# Show README with pager for long content
git-remote-color -d

# Show README with full output (no pager)
git-remote-color -d -f

# Alternative flags
git-remote-color --detail
git-remote-color --readme

# Combine with path
git-remote-color -d ../other-project

# Pipe to file (use -f to avoid pager)
git-remote-color -df > output.txt
```

### Help
```bash
git-remote-color -h
git-remote-color --help
```

---

## 🎯 Example Output

### Basic Output
```text
origin  https://github.com/user/repo (fetch, push)
   A powerful CLI tool
   🌍 public  ⭐ 42  🍴 10  🐞 3  ⬇ 2500  🕒 2024-01-15
   🧠 Go 70.0%, Shell 20.0%, Makefile 10.0%

   🌿 branches:
     - main ★
     - dev

   🏷️ tags:
     - v1.0
     - v1.1
```

### With README (-d flag)
```text
origin  https://github.com/user/repo (fetch, push)
   A powerful CLI tool
   🌍 public  ⭐ 42  🍴 10  🐞 3  ⬇ 2500  🕒 2024-01-15
   🧠 Go 70.0%, Shell 20.0%, Makefile 10.0%

   🌿 branches:
     - main ★
     - dev

════ README ═══
   📄 README.md
   ────────────────────────────────────────────────────────────

   # Project Title
   
   A beautiful README rendered with Glamour...
   
   ## Installation
   
   ```bash
   go install github.com/user/repo@latest
   ```
   
   ... (opens in pager for long content)
```

---

## ⚙️ Configuration

Auto-detected config file (JSON):

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
  "github_token": "",
  "glamour_style": "auto",
  "glamour_width": 100,
  "language_colors": [
    "#FF5555",
    "#55FF55",
    "#5599FF",
    "#FFFF55",
    "#FF55FF",
    "#55FFFF",
    "#FFA500"
  ]
}
```

### New Configuration Options

| Field | Description | Default |
|-------|-------------|---------|
| `glamour_style` | README rendering style | `"auto"` |
| `glamour_width` | Word wrap width for README | `100` |
| `readme_color` | Color for README title | `"#95E1D3"` |
| `language_colors` | Custom language color palette | Rainbow array |

#### Glamour Styles
- `auto` - Auto-detect based on terminal background
- `light` - Light theme
- `dark` - Dark theme
- `notty` - No TTY style (plain)

Config lookup order:
* `GIT_REMOTE_COLOR_CONFIG` environment variable
* Executable-side config (`gitv.json`, `git-remote-color.json`)
* Current working directory
* Platform config directory
  * Windows: `%AppData%`
  * Linux: `$XDG_CONFIG_HOME` or `~/.config`
  * macOS: `~/Library/Application Support`
* Home directory dotfiles (`.gitv.json`, `.git-remote-color.json`)

---

## 🎨 Color Fields

| Field          | Description          |
| -------------- | -------------------- |
| remote         | Remote name          |
| scheme         | Protocol             |
| host           | Domain               |
| path           | Username/org         |
| repo           | Repo name            |
| fetch          | fetch label          |
| push           | push label           |
| description    | repo description     |
| branch         | branch names         |
| tag            | tag names            |
| visibility     | public/private label |
| last_update    | last update date     |
| readme_color   | README title color   |

---

## 📦 Installation

### From Source
```bash
go install github.com/cumulus13/git-remote-color@latest
```

### From Releases
Download the latest binary from [Releases](https://github.com/cumulus13/git-remote-color/releases) page.

Available for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

---

## 🔧 Dependencies

- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- Go 1.20+

---

## ⚠️ Notes

* GitHub API used for metadata
* Without token → 60 req/hour
* With token → 5000 req/hour
* Private repos require token
* README fetch is a separate API call
* Pager respects `$PAGER` environment variable

---

## ❌ Non-GitHub Repos

* Still shown (colored)
* No metadata or README fetched
* Works with any Git remote

---

## 🧠 Behavior Summary

| Scenario             | Result         |
| -------------------- | -------------- |
| Same fetch/push      | grouped        |
| Different fetch/push | both shown     |
| Multiple remotes     | all shown      |
| Subfolder            | works          |
| Offline              | cache fallback |
| No cache + offline   | warning        |
| README > 50 lines    | opens in pager |
| README + `-f` flag   | direct output  |
| No README            | friendly msg   |

---

## 🗺️ Roadmap

- [ ] Support for GitLab, Bitbucket APIs
- [ ] Config file generator command
- [ ] Shell completion scripts
- [ ] Multiple output formats (JSON, YAML)
- [ ] Watch mode for live updates

---

## 📄 License

MIT

---

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

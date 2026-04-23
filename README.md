# git-remote-color

> A **colorized, GitHub-aware replacement for `git remote -v`** with rich metadata, smart caching, and offline support.

<p align="center">
  <img alt="Release" src="https://img.shields.io/github/v/release/cumulus13/git-remote-color?color=blue">
  <img alt="Downloads" src="https://img.shields.io/github/downloads/cumulus13/git-remote-color/total">
  <img alt="License" src="https://img.shields.io/github/license/cumulus13/git-remote-color">
  <img alt="Go Version" src="https://img.shields.io/badge/go-1.20+-00ADD8?logo=go">
</p>

---

## ✨ Features

### 🎨 Rich Colored Output

* Truecolor ANSI (24-bit)
* Fully configurable via JSON
* Clean, structured CLI layout

---

### 🔗 Git Smart Integration

* Works like `git remote -v`
* Supports:

  * current directory
  * subdirectories inside repo
  * relative path
  * absolute path
* Automatically detects Git root

---

### 🌐 GitHub Deep Info

For GitHub repositories:

* 📝 Description
* 🌍 Public / 🔒 Private
* ⭐ Stars / 🍴 Forks / 🐞 Issues
* 🧠 **Languages with percentage (sorted)**
* 🌿 Branch list
* 🏷️ Tag list

---

### 🧠 Language Breakdown (Improved)

Instead of a single language:

```text
🧠 JavaScript 82.4%, HTML 12.1%, CSS 5.5%
```

* Sorted by usage (descending)
* Based on GitHub language API
* Accurate percentage calculation

---

### ⚡ Smart Cache System

* Automatic in-memory cache (1 hour TTL)
* Prevents duplicate API calls
* Faster repeated runs

#### Cache behavior:

| Scenario  | Behavior               |
| --------- | ---------------------- |
| First run | Fetch from GitHub      |
| Next runs | Use cache              |
| Offline   | Use cache if available |

---

### 📡 Offline Support

When internet is unavailable:

#### ✔ With cache:

```text
(cached)
🌍 public ⭐ 10 🍴 2 🐞 1
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
| Public repo  | No token used         |
| Private repo | Uses token            |
| Rate limit   | Auto retry with token |

---

## 🚀 Usage

```bash
git remote-color
git remote-color ../project
git remote-color C:\PROJECTS\repo
```

---

## 🎯 Example Output

```text
origin  https://github.com/user/repo (fetch, push)
   A powerful CLI tool
   🌍 public  ⭐ 42  🍴 10  🐞 3
   🧠 Go 70.0%, Shell 20.0%, Makefile 10.0%

   🌿 branches:
     - main
     - dev

   🏷️ tags:
     - v1.0
     - v1.1
```

---

## ⚙️ Configuration

Auto-created config:

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
  "github_token": ""
}
```

---

## 🎨 Color Fields

| Field       | Description          |
| ----------- | -------------------- |
| remote      | Remote name          |
| scheme      | Protocol             |
| host        | Domain               |
| path        | Username/org         |
| repo        | Repo name            |
| fetch       | fetch label          |
| push        | push label           |
| description | repo description     |
| branch      | branch names         |
| tag         | tag names            |
| visibility  | public/private label |

---

## ⚠️ Notes

* GitHub API used for metadata
* Without token → 60 req/hour
* With token → 5000 req/hour
* Private repos require token

---

## ❌ Non-GitHub Repos

* Still shown (colored)
* No metadata fetched

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

---

## 📄 License

MIT

---

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)
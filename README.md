# git-remote-color

Colorized and enhanced `git remote -v` with parsing, and icons

## ✨ Features

- 🎨 Truecolor + 256 fallback
- 🔐 SSH / 🌐 HTTPS detection
- ⚙️ Configurable colors (JSON/YAML/TOML)
- 📦 Git subcommand (`git remote-color`)
- 📤 JSON output

---

## 🚀 Install

### Go
```bash
go install github.com/cumulus13/git-remote-color@latest
````

### Scoop (Windows)

```bash
scoop install git-remote-color
```

### Homebrew (macOS)

```bash
brew install git-remote-color
```

---

## 🧪 Usage

```bash
git remote-color
git remote-color ../repo
git remote-color --json
```

---

## ⚙️ Config

Auto-generated config:

```json
{
  "remote": "#00FFFF",
  "scheme": "#FFAAFF",
  "host": "#55AA00",
  "path": "#AAAAFF",
  "repo": "#FFFF00",
  "fetch": "#00AAFF",
  "push": "#AA5500"
}
```

---

## 📄 License

MIT

## 👤 Author
        
[Hadi Cahyadi](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)
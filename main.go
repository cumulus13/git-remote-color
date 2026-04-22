package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/sys/windows"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Remote string `json:"remote" yaml:"remote" toml:"remote"`
	Scheme string `json:"scheme" yaml:"scheme" toml:"scheme"`
	Host   string `json:"host" yaml:"host" toml:"host"`
	Path   string `json:"path" yaml:"path" toml:"path"`
	Repo   string `json:"repo" yaml:"repo" toml:"repo"`
	Fetch  string `json:"fetch" yaml:"fetch" toml:"fetch"`
	Push   string `json:"push" yaml:"push" toml:"push"`
}

type Output struct {
	Remote string `json:"remote"`
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	User   string `json:"user"`
	Repo   string `json:"repo"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}

var defaultConfig = Config{
	Remote: "#00FFFF",
	Scheme: "#FFAAFF",
	Host:   "#55AA00",
	Path:   "#AAAAFF",
	Repo:   "#FFFF00",
	Fetch:  "#00AAFF",
	Push:   "#AA5500",
}

var noColor bool
var jsonMode bool
var colorMode = "truecolor"

// ---------- TERMINAL ----------

func enableWindowsANSI() {
	if runtime.GOOS != "windows" {
		return
	}
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)
}

func detectColorSupport() {
	if noColor || os.Getenv("NO_COLOR") != "" {
		colorMode = "none"
		return
	}
	if strings.Contains(os.Getenv("COLORTERM"), "truecolor") {
		colorMode = "truecolor"
		return
	}
	if strings.Contains(os.Getenv("TERM"), "256") {
		colorMode = "256"
		return
	}
	if runtime.GOOS == "windows" {
		colorMode = "truecolor"
		return
	}
	colorMode = "none"
}

// ---------- COLOR ----------

func hexToRGB(hex string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func rgbTo256(r, g, b int) int {
	return 16 + 36*(r/51) + 6*(g/51) + (b / 51)
}

func colorize(hex, text string) string {
	if colorMode == "none" {
		return text
	}
	r, g, b := hexToRGB(hex)
	if colorMode == "truecolor" {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
	}
	code := rgbTo256(r, g, b)
	return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m", code, text)
}

// ---------- CONFIG ----------

func resolveConfigPath() (string, string) {
	exe, _ := os.Executable()
	base := strings.TrimSuffix(exe, filepath.Ext(exe))
	for _, ext := range []string{".json", ".yaml", ".yml", ".toml"} {
		p := base + ext
		if _, err := os.Stat(p); err == nil {
			return p, ext
		}
	}
	return base + ".json", ".json"
}

func saveDefault(path string) {
	data, _ := json.MarshalIndent(defaultConfig, "", "  ")
	os.WriteFile(path, data, 0644)
}

func loadConfig() Config {
	path, ext := resolveConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		saveDefault(path)
		return defaultConfig
	}
	data, _ := os.ReadFile(path)
	cfg := defaultConfig
	switch ext {
	case ".json":
		json.Unmarshal(data, &cfg)
	case ".yaml", ".yml":
		yaml.Unmarshal(data, &cfg)
	case ".toml":
		toml.Unmarshal(data, &cfg)
	}
	return cfg
}

// ---------- ICON ----------

func detectIcon(scheme string) string {
	switch {
	case strings.Contains(scheme, "ssh"):
		return "🔐"
	case strings.Contains(scheme, "http"):
		return "🌐"
	case strings.Contains(scheme, "git"):
		return "📦"
	default:
		return "📁"
	}
}

// ---------- MAIN ----------

func main() {
	for _, a := range os.Args {
		if a == "--no-color" {
			noColor = true
		}
		if a == "--json" {
			jsonMode = true
		}
	}

	enableWindowsANSI()
	detectColorSupport()
	cfg := loadConfig()

	// path
	dir := "."
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		dir = os.Args[1]
	}

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	out, _ := cmd.Output()

	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`(?P<scheme>\w+://)?(?P<host>[^/:@]+)[:/](?P<path>.+)`)

	var results []Output

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}

		remote, url := f[0], f[1]
		extra := ""
		if len(f) > 2 {
			extra = f[2]
		}

		var scheme, host, path string

		if strings.Contains(url, "@") && strings.Contains(url, ":") && !strings.Contains(url, "://") {
			p := strings.Split(url, "@")[1]
			hp := strings.SplitN(p, ":", 2)
			host, path = hp[0], hp[1]
			scheme = "ssh://"
		} else {
			m := re.FindStringSubmatch(url)
			if m != nil {
				scheme, host, path = m[1], m[2], m[3]
			}
		}

		pp := strings.Split(path, "/")
		user, repo := "", ""
		if len(pp) >= 2 {
			user = pp[len(pp)-2]
			repo = pp[len(pp)-1]
		}

		if jsonMode {
			results = append(results, Output{
				Remote: remote,
				Scheme: scheme,
				Host:   host,
				User:   user,
				Repo:   repo,
				Type:   extra,
				URL:    url,
			})
			continue
		}

		icon := detectIcon(scheme)

		fmt.Printf("%-8s  %s%s%s/%s/%s",
			colorize(cfg.Remote, remote),
			icon+" ",
			colorize(cfg.Scheme, scheme),
			colorize(cfg.Host, host),
			colorize(cfg.Path, user),
			colorize(cfg.Repo, repo),
		)

		if extra != "" {
			if strings.Contains(extra, "fetch") {
				fmt.Print(" " + colorize(cfg.Fetch, extra))
			} else if strings.Contains(extra, "push") {
				fmt.Print(" " + colorize(cfg.Push, extra))
			} else {
				fmt.Print(" " + extra)
			}
		}

		fmt.Println()
	}

	if jsonMode {
		j, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(j))
	}
}
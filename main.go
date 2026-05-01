package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
)

type LangMap map[string]int

type Config struct {
	Remote         string   `json:"remote"`
	Scheme         string   `json:"scheme"`
	Host           string   `json:"host"`
	Path           string   `json:"path"`
	Repo           string   `json:"repo"`
	Fetch          string   `json:"fetch"`
	Push           string   `json:"push"`
	Description    string   `json:"description"`
	Branch         string   `json:"branch"`
	Tag            string   `json:"tag"`
	Token          string   `json:"github_token"`
	Visibility     string   `json:"visibility"`
	LastUpdate     string   `json:"last_update"`
	LanguageColors []string `json:"language_colors"`
	ReadmeColor    string   `json:"readme_color"`
	GlamourStyle   string   `json:"glamour_style"`
	GlamourWidth   int      `json:"glamour_width"`
}

type Row struct {
	Remote string
	URL    string
	Type   string
	Host   string
	User   string
	Repo   string
	Scheme string
}

type GitHubRepo struct {
	Description   string `json:"description"`
	Language      string `json:"language"`
	Stars         int    `json:"stargazers_count"`
	Forks         int    `json:"forks_count"`
	Issues        int    `json:"open_issues_count"`
	Private       bool   `json:"private"`
	UpdatedAt     string `json:"updated_at"`
	DefaultBranch string `json:"default_branch"`
}

type Release struct {
	Assets []struct {
		DownloadCount int `json:"download_count"`
	} `json:"assets"`
}

type Branch struct {
	Name string `json:"name"`
}
type Tag struct {
	Name string `json:"name"`
}

type CacheEntry struct {
	Repo      GitHubRepo
	Branches  []string
	Tags      []string
	Languages map[string]float64
	Downloads int
	Time      int64
	Cached    bool
}

type HTTPError struct {
	Status int
}

type ReadmeInfo struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type Args struct {
	Dir        string
	Detail     bool
	Help       bool
	FullOutput bool // -f flag: disable pager, output directly
}

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	cache      = map[string]CacheEntry{}
	mu         sync.Mutex
)

var defaultLangColors = []string{
	"#FF5555", // red
	"#55FF55", // green
	"#5599FF", // blue
	"#FFFF55", // yellow
	"#FF55FF", // magenta
	"#55FFFF", // cyan
	"#FFA500", // orange
}

// ---------- COLOR ----------
func color(hex, text string) string {
	if hex == "" {
		return text
	}
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

func getLangColors(cfg Config) []string {
	if len(cfg.LanguageColors) > 0 {
		return cfg.LanguageColors
	}
	return defaultLangColors
}

// ---------- HELP ----------
func printHelp() {
	cfg := defaultConfig()
	
	help := `
` + color("#FF6B6B", "🔧 git-remote-color") + ` - Beautiful Git Remote Information

` + color("#FFE66D", "USAGE:") + `
  git-remote-color [OPTIONS] [DIRECTORY]

` + color("#4ECDC4", "ARGUMENTS:") + `
  DIRECTORY               ` + color("#888888", "(optional)") + `  Path to git repository
                          Can be:
                          • ` + color(cfg.Remote, ".") + `                Current directory (default)
                          • ` + color(cfg.Remote, "relative/path") + `    Relative path
                          • ` + color(cfg.Remote, "/absolute/path") + `   Absolute path
                          • ` + color(cfg.Remote, "~") + `               Home directory
                          • ` + color(cfg.Remote, "..") + `              Parent directory

` + color("#4ECDC4", "FLAGS:") + `
  ` + color(cfg.Description, "-d, --detail") + `    Show README from remote repository (uses pager)
  ` + color(cfg.Description, "-r, --readme") + `    Same as --detail
  ` + color(cfg.Description, "-f, --full") + `      Disable pager, print all output directly
  ` + color(cfg.Description, "-h, --help") + `      Show this help message

` + color("#95E1D3", "EXAMPLES:") + `
  # Show info for current directory
  ` + color(cfg.Repo, "git-remote-color") + `

  # Show info with README (pager mode for long content)
  ` + color(cfg.Repo, "git-remote-color -d") + `

  # Show info with full README output (no pager)
  ` + color(cfg.Repo, "git-remote-color -d -f") + `

  # Pipe the output to a file
  ` + color(cfg.Repo, "git-remote-color -df > output.txt") + `

  # Show info for specific repo with README
  ` + color(cfg.Repo, "git-remote-color --detail /path/to/repo") + `

` + color("#F38181", "PAGER BEHAVIOR:") + `
  By default, long README content opens in a pager (less -R).
  • Use ` + color(cfg.Tag, "↑/↓") + ` or ` + color(cfg.Tag, "j/k") + ` to scroll
  • Press ` + color(cfg.Tag, "q") + ` to quit the pager
  • Use ` + color(cfg.Tag, "-f") + ` to print directly without pager
  • Pager preserves all colors and formatting ✨

` + color("#FEDE5D", "CONFIGURATION:") + `
  Config file is auto-detected from:
  • Current directory
  • Executable directory
  • User config directory (~/.config/git-remote-color/)
  • Home directory
  
  Supported names: ` + color(cfg.Tag, "gitv.json, giti.json, git-remote-color.json") + `

` + color("#FEDE5D", "ENVIRONMENT:") + `
  GIT_REMOTE_COLOR_CONFIG    Path to custom config file
  GITHUB_TOKEN               GitHub API token (use in config file's "github_token" field)
`
	fmt.Println(help)
}

// ---------- DEFAULT CONFIG ----------
func defaultConfig() Config {
	return Config{
		Remote: "#00FFFF", Scheme: "#FFAAFF", Host: "#55AA00",
		Path: "#AAAAFF", Repo: "#FFFF00",
		Fetch: "#00AAFF", Push: "#AA5500",
		Description: "#00AAFF",
		Branch:      "#FFAAFF", Tag: "#AAAA00",
		Visibility:  "#00FFFF",
		LastUpdate:  "#FFFF00",
		ReadmeColor: "#95E1D3",
		GlamourStyle: "auto",
		GlamourWidth: 100,
	}
}

// ---------- PARSE ARGUMENTS ----------
func parseArgs() Args {
	args := Args{
		Dir:        ".",
		Detail:     false,
		Help:       false,
		FullOutput: false,
	}
	
	foundDir := false
	
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		
		switch {
		case arg == "-h" || arg == "--help":
			args.Help = true
			return args
			
		case arg == "-d" || arg == "--detail" || arg == "-r" || arg == "--readme":
			args.Detail = true
			
		case arg == "-f" || arg == "--full":
			args.FullOutput = true
			
		default:
			// Accept any non-flag argument as directory path
			if !strings.HasPrefix(arg, "-") && !foundDir {
				args.Dir = arg
				foundDir = true
			}
		}
	}
	
	return args
}

// ---------- LOAD CONFIG ----------
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func configCandidates() []string {
	exe, _ := os.Executable()
	exeBase := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	exeDir := filepath.Dir(exe)

	names := uniqueStrings([]string{
		exeBase + ".json",
		"gitv.json",
		"giti.json",
		"git-remote-color.json",
		".gitv.json",
		".giti.json",
		".git-remote-color.json",
	})

	var candidates []string

	if env := strings.TrimSpace(os.Getenv("GIT_REMOTE_COLOR_CONFIG")); env != "" {
		candidates = append(candidates, env)
	}

	for _, name := range names {
		candidates = append(candidates, filepath.Join(exeDir, name))
	}

	if cwd, err := os.Getwd(); err == nil {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(cwd, name))
		}
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(configDir, "git-remote-color", name),
				filepath.Join(configDir, exeBase, name),
				filepath.Join(configDir, name),
			)
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(home, name))
		}
	}

	if runtime.GOOS != "windows" {
		if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
			for _, name := range names {
				candidates = append(candidates,
					filepath.Join(xdg, "git-remote-color", name),
					filepath.Join(xdg, exeBase, name),
				)
			}
		}
	}

	return uniqueStrings(candidates)
}

func loadConfig() Config {
	cfg := defaultConfig()

	for _, path := range configCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		_ = json.Unmarshal(data, &cfg)
		break
	}

	return cfg
}

// ---------- FIND GIT ROOT ----------
func findGitRoot(start string) (string, error) {
	dir := start

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a git repository")
}

// ---------- PARSER ----------
func parse(line string) (Row, bool) {
	f := strings.Fields(line)
	if len(f) < 2 {
		return Row{}, false
	}

	r := Row{Remote: f[0], URL: f[1]}
	if len(f) > 2 {
		r.Type = f[2]
	}

	url := r.URL

	// SSH
	if strings.Contains(url, "@") && strings.Contains(url, ":") && !strings.Contains(url, "://") {
		right := strings.Split(url, "@")[1]
		hp := strings.SplitN(right, ":", 2)
		r.Host = hp[0]
		r.Scheme = "ssh://"
		pp := strings.Split(hp[1], "/")
		r.User = pp[0]
		r.Repo = strings.TrimSuffix(pp[1], ".git")
		return r, true
	}

	// HTTPS
	if strings.Contains(url, "://") {
		u := strings.SplitN(url, "://", 2)
		r.Scheme = u[0] + "://"
		parts := strings.Split(u[1], "/")
		if len(parts) >= 3 {
			r.Host = parts[0]
			r.User = parts[1]
			r.Repo = strings.TrimSuffix(parts[2], ".git")
			return r, true
		}
	}

	return r, false
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("http %d", e.Status)
}

// ---------- HTTP ----------
func getJSON(url, token string, target interface{}) (int, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "git-remote-color")
	req.Header.Set("Accept", "application/vnd.github+json")
	if token = strings.TrimSpace(token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return resp.StatusCode, HTTPError{Status: resp.StatusCode}
	}

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, json.Unmarshal(body, target)
}

func formatDate(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// ---------- FETCH README ----------
func fetchReadme(user, repo, token string) *ReadmeInfo {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", user, repo)
	
	var readme ReadmeInfo
	status, err := getJSON(url, token, &readme)
	
	if err != nil || status != 200 {
		return nil
	}
	
	return &readme
}

func decodeBase64(s string) (string, error) {
	// Remove newlines and whitespace from base64 content
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, s)
	
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	
	return string(decoded), nil
}

// ---------- PAGER ----------
func showInPager(content string) {
	// Try less first (most common), fallback to more
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}
	
	cmd := exec.Command(pager, "-R") // -R preserves ANSI color codes
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		// Fallback: try more if less not available
		if pager == "less" {
			cmd = exec.Command("more")
			cmd.Stdin = strings.NewReader(content)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				// Last resort: just print it
				fmt.Print(content)
			}
		} else {
			// Custom pager failed, print directly
			fmt.Print(content)
		}
	}
}

// ---------- SHOW README ----------
func showReadme(user, repo string, cfg Config, fullOutput bool) {
	fmt.Println("\n" + color("#FF6B6B", "═══ README ═══"))
	readme := fetchReadme(user, repo, cfg.Token)
	
	if readme != nil {
		fmt.Println("   " + color(cfg.ReadmeColor, "📄 "+readme.Name))
		fmt.Println("   " + color("#888888", strings.Repeat("─", 60)))
		fmt.Println()
		
		if readme.Encoding == "base64" {
			decoded, err := decodeBase64(readme.Content)
			if err == nil {
				// Configure glamour renderer
				width := cfg.GlamourWidth
				if width == 0 {
					width = 100
				}
				
				style := cfg.GlamourStyle
				if style == "" {
					style = "auto"
				}
				
				var renderer *glamour.TermRenderer
				if style == "auto" {
					renderer, err = glamour.NewTermRenderer(
						glamour.WithAutoStyle(),
						glamour.WithWordWrap(width),
					)
				} else {
					renderer, err = glamour.NewTermRenderer(
						glamour.WithStandardStyle(style),
						glamour.WithWordWrap(width),
					)
				}
				
				if err != nil {
					fmt.Println("   " + color("#FF5555", "⚠ Could not create renderer"))
					return
				}
				
				rendered, err := renderer.Render(decoded)
				if err != nil {
					fmt.Println("   " + color("#FF5555", "⚠ Could not render README"))
					return
				}
				
				// Check content length
				lines := strings.Split(rendered, "\n")
				
				if fullOutput || len(lines) <= 50 {
					// Print directly (either forced with -f or short content)
					fmt.Print(rendered)
				} else {
					// Use pager for long content
					fmt.Println(color("#888888", "   📖 Opening in pager (use -f for direct output, q to quit)"))
					showInPager(rendered)
				}
			} else {
				fmt.Println("   " + color("#FF5555", "⚠ Could not decode README"))
			}
		} else {
			// Non-base64 content (unlikely, but handle it)
			lines := strings.Split(readme.Content, "\n")
			if fullOutput || len(lines) <= 50 {
				fmt.Print(readme.Content)
			} else {
				fmt.Println(color("#888888", "   📖 Opening in pager (use -f for direct output, q to quit)"))
				showInPager(readme.Content)
			}
		}
	} else {
		fmt.Println("   " + color("#FFE66D", "⚠ No README found in this repository"))
		fmt.Println("   " + color("#888888", "  The repository might exist but doesn't have a README file"))
	}
}

// ---------- FETCH ----------
func fetchAll(user, repo, token string) CacheEntry {
	key := user + "/" + repo

	mu.Lock()
	if c, ok := cache[key]; ok && time.Now().Unix()-c.Time < 3600 {
		mu.Unlock()
		c.Cached = true
		return c
	}
	mu.Unlock()

	entry := CacheEntry{}

	status, err := getJSON("https://api.github.com/repos/"+user+"/"+repo, token, &entry.Repo)

	if err != nil {
		if status != 0 {
			msg := fmt.Sprintf("❌ repo error (HTTP %d)", status)

			if status == 404 {
				msg = "❌ repo not found"
			} else if status == 403 {
				msg = "🔒 access denied"
			} else if status == 401 {
				msg = "🔑 invalid github token"
			}

			return CacheEntry{
				Repo: GitHubRepo{
					Description: msg,
				},
			}
		}

		mu.Lock()
		if c, ok := cache[key]; ok {
			mu.Unlock()
			c.Cached = true
			return c
		}
		mu.Unlock()

		return CacheEntry{
			Repo: GitHubRepo{
				Description: "⚠ offline (no cache available)",
			},
		}
	}

	var releases []Release
	getJSON("https://api.github.com/repos/"+user+"/"+repo+"/releases", token, &releases)

	totalDownloads := 0
	for _, r := range releases {
		for _, a := range r.Assets {
			totalDownloads += a.DownloadCount
		}
	}

	entry.Downloads = totalDownloads

	var branches []Branch
	getJSON("https://api.github.com/repos/"+user+"/"+repo+"/branches", token, &branches)
	for _, b := range branches {
		entry.Branches = append(entry.Branches, b.Name)
	}

	var tags []Tag
	getJSON("https://api.github.com/repos/"+user+"/"+repo+"/tags", token, &tags)
	for _, t := range tags {
		entry.Tags = append(entry.Tags, t.Name)
	}

	entry.Time = time.Now().Unix()

	var langRaw LangMap
	getJSON("https://api.github.com/repos/"+user+"/"+repo+"/languages", token, &langRaw)

	total := 0
	for _, v := range langRaw {
		total += v
	}

	entry.Languages = map[string]float64{}

	if total > 0 {
		for k, v := range langRaw {
			entry.Languages[k] = (float64(v) / float64(total)) * 100
		}
	}

	mu.Lock()
	cache[key] = entry
	mu.Unlock()

	return entry
}

// ---------- MAIN ----------
func main() {
	args := parseArgs()
	
	if args.Help {
		printHelp()
		return
	}
	
	cfg := loadConfig()

	// Handle home directory expansion
	dir := args.Dir
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, dir[1:])
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println(color("#FF5555", "❌ Error:"), "Invalid path:", dir)
		return
	}

	root, err := findGitRoot(absDir)
	if err != nil {
		fmt.Println(color("#FF5555", "❌"), "Not a git repository:", absDir)
		fmt.Println(color("#888888", "   Tip: Run 'git init' or navigate to a git repository"))
		return
	}

	// Show which repo we're looking at if not current directory
	if dir != "." {
		fmt.Println(color("#888888", "📂 Repository:"), color(cfg.Path, root))
	}

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = root

	out, err := cmd.Output()
	if err != nil {
		fmt.Println(color("#FF5555", "❌ Error:"), err)
		return
	}

	group := map[string][]Row{}

	for _, l := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}

		r, ok := parse(l)
		if !ok {
			continue
		}

		group[r.URL] = append(group[r.URL], r)
	}

	if len(group) == 0 {
		fmt.Println(color("#FFE66D", "⚠ No remote repositories configured"))
		fmt.Println(color("#888888", "   Add a remote with: git remote add origin <url>"))
		return
	}

	for _, rows := range group {
		r := rows[0]

		line := color(cfg.Remote, r.Remote) + "  " +
			color(cfg.Scheme, r.Scheme) +
			color(cfg.Host, r.Host) + "/" +
			color(cfg.Path, r.User) + "/" +
			color(cfg.Repo, r.Repo)

		var types []string
		for _, rr := range rows {
			if strings.Contains(rr.Type, "fetch") {
				types = append(types, color(cfg.Fetch, "fetch"))
			}
			if strings.Contains(rr.Type, "push") {
				types = append(types, color(cfg.Push, "push"))
			}
		}
		if len(types) > 0 {
			line += " (" + strings.Join(types, ", ") + ")"
		}

		fmt.Println(line)

		if r.Host == "github.com" {
			data := fetchAll(r.User, r.Repo, cfg.Token)

			if data.Cached {
				fmt.Println("   " + color("#888888", "(cached)"))
			}

			if data.Repo.Description == "⚠ offline (no cache available)" {
				fmt.Println("   " + color("#FF5555", "⚠ offline (no cached data)"))
				continue
			}

			// Handle error states
			if strings.HasPrefix(data.Repo.Description, "❌") || 
			   strings.HasPrefix(data.Repo.Description, "🔒") || 
			   strings.HasPrefix(data.Repo.Description, "🔑") {
				fmt.Println("   " + color("#FF5555", data.Repo.Description))
				continue
			}

			if data.Repo.Description != "" {
				fmt.Println("   " + color(cfg.Description, data.Repo.Description))
			}

			visibility := "🌍 public"
			if data.Repo.Private {
				visibility = "🔒 private"
			}

			fmt.Printf("   %s  ⭐ %d  🍴 %d  🐞 %d  ⬇ %d  🕒 %s\n",
				color(cfg.Visibility, visibility),
				data.Repo.Stars,
				data.Repo.Forks,
				data.Repo.Issues,
				data.Downloads,
				color(cfg.LastUpdate, formatDate(data.Repo.UpdatedAt)),
			)

			// Languages
			if len(data.Languages) > 0 {
				type langPair struct {
					Name string
					Pct  float64
				}

				var langs []langPair
				for k, v := range data.Languages {
					langs = append(langs, langPair{k, v})
				}

				sort.Slice(langs, func(i, j int) bool {
					return langs[i].Pct > langs[j].Pct
				})

				colors := getLangColors(cfg)

				var parts []string
				for i, l := range langs {
					c := colors[i%len(colors)]
					parts = append(parts,
						color(c, fmt.Sprintf("%s %.1f%%", l.Name, l.Pct)),
					)
				}

				fmt.Println("   🧠", strings.Join(parts, ", "))
			}

			// Branches
			if len(data.Branches) > 0 {
				fmt.Println("   🌿 branches:")
				for _, b := range data.Branches {
					marker := ""
					if b == data.Repo.DefaultBranch {
						marker = color("#FFD700", " ★")
					}
					fmt.Println("     -", color(cfg.Branch, b)+marker)
				}
			}

			// Tags
			if len(data.Tags) > 0 {
				fmt.Println("   🏷️ tags:")
				for _, t := range data.Tags {
					fmt.Println("     -", color(cfg.Tag, t))
				}
			}
			
			// Show README if requested
			if args.Detail {
				showReadme(r.User, r.Repo, cfg, args.FullOutput)
			}
		}
	}
}
package main

import (
	"context"
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

// ─── Version ──────────────────────────────────────────────────────────────────

const Version = "1.1.0"

// ─── Types ────────────────────────────────────────────────────────────────────

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
	WorkflowColor  string   `json:"workflow_color"`
	// Owner used to resolve bare repo names (e.g. "myrepo" → owner/myrepo)
	Owner string `json:"owner"`
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

// WorkflowRun represents a single GitHub Actions workflow run.
type WorkflowRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`     // queued, in_progress, completed
	Conclusion string `json:"conclusion"` // success, failure, cancelled, skipped, neutral, timed_out, action_required, stale
	UpdatedAt  string `json:"updated_at"`
	HTMLURL    string `json:"html_url"`
	HeadBranch string `json:"head_branch"`
	Event      string `json:"event"`
}

// WorkflowRunsResponse wraps the GitHub /actions/runs endpoint.
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

type CacheEntry struct {
	Repo         GitHubRepo
	Branches     []string
	Tags         []string
	Languages    map[string]float64
	Downloads    int
	WorkflowRuns []WorkflowRun
	Time         int64
	Cached       bool
}

type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("HTTP %d", e.Status)
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
	FullOutput bool
	Version    bool
	Workflows  bool
	NoCache    bool
	Command    string // "config"
	Subcommand string // "show" or "path"
}

// ─── Globals ──────────────────────────────────────────────────────────────────

var (
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	cache   = map[string]CacheEntry{}
	cacheMu sync.RWMutex
)

const cacheTTL = time.Hour

var defaultLangColors = []string{
	"#FF5555", "#55FF55", "#5599FF", "#FFFF55",
	"#FF55FF", "#55FFFF", "#FFA500",
}

// ─── Color ────────────────────────────────────────────────────────────────────

func color(hex, text string) string {
	if hex == "" || text == "" {
		return text
	}
	hex = strings.TrimSpace(hex)
	if !strings.HasPrefix(hex, "#") || (len(hex) != 7 && len(hex) != 4) {
		return text
	}
	// Expand short form #RGB → #RRGGBB
	if len(hex) == 4 {
		hex = "#" + string([]byte{hex[1], hex[1], hex[2], hex[2], hex[3], hex[3]})
	}
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return text
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

func getLangColors(cfg Config) []string {
	if len(cfg.LanguageColors) > 0 {
		return cfg.LanguageColors
	}
	return defaultLangColors
}

// ─── Help / Version ───────────────────────────────────────────────────────────

func printVersion() {
	fmt.Printf("git-remote-color v%s\n", Version)
}

func printHelp() {
	cfg := defaultConfig()

	fmt.Print(`
` + color("#FF6B6B", "🔧 git-remote-color v"+Version) + ` — Beautiful Git Remote Information

` + color("#FFE66D", "USAGE:") + `
  git-remote-color [OPTIONS] [DIRECTORY|REPO]

` + color("#4ECDC4", "ARGUMENTS:") + `
  DIRECTORY / REPO        ` + color("#888888", "(optional)") + `  Path or repo name
                          • ` + color(cfg.Repo, ".") + `                  Current directory (default)
                          • ` + color(cfg.Repo, "relative/path") + `      Relative path
                          • ` + color(cfg.Repo, "/absolute/path") + `     Absolute path
                          • ` + color(cfg.Repo, "~") + `                 Home directory
                          • ` + color(cfg.Repo, "myrepo") + `             Resolved via "owner" in config
                          • ` + color(cfg.Repo, "owner/repo") + `         Direct GitHub slug

` + color("#4ECDC4", "FLAGS:") + `
  ` + color(cfg.Description, "-d, --detail") + `    Show README from remote repository
  ` + color(cfg.Description, "-r, --readme") + `    Same as --detail
  ` + color(cfg.Description, "-w, --workflows") + ` Show GitHub Actions workflow status
  ` + color(cfg.Description, "-f, --full") + `      Disable pager, print all output directly
  ` + color(cfg.Description, "--no-cache") + `      Bypass in-memory cache
  ` + color(cfg.Description, "-v, --version") + `   Show version
  ` + color(cfg.Description, "-h, --help") + `      Show this help message

` + color("#FEDE5D", "config:") + `
  config show             Show the active, parsed JSON configuration
  config path             List all config file search locations (active files highlighted)

` + color("#95E1D3", "EXAMPLES:") + `
  ` + color(cfg.Repo, "git-remote-color") + `                     Info for current dir
  ` + color(cfg.Repo, "git-remote-color -d") + `                  Info + README (pager)
  ` + color(cfg.Repo, "git-remote-color -d -f") + `               Info + README (no pager)
  ` + color(cfg.Repo, "git-remote-color -w") + `                  Info + workflow status
  ` + color(cfg.Repo, "git-remote-color -dw") + `                 Info + README + workflows
  ` + color(cfg.Repo, "git-remote-color myrepo") + `              Resolve via config owner
  ` + color(cfg.Repo, "git-remote-color owner/repo") + `          Direct slug lookup
  ` + color(cfg.Repo, "git-remote-color -d ../other") + `         Specific path + README
  ` + color(cfg.Repo, "git-remote-color -df > out.txt") + `       Pipe to file

` + color("#F38181", "PAGER BEHAVIOR:") + `
  Long README content opens in a pager (` + color(cfg.Tag, "$PAGER") + ` or ` + color(cfg.Tag, "less -R") + `).
  • ` + color(cfg.Tag, "↑/↓") + ` or ` + color(cfg.Tag, "j/k") + `   scroll   • ` + color(cfg.Tag, "q") + ` quit   • ` + color(cfg.Tag, "-f") + ` direct output

` + color("#FEDE5D", "CONFIGURATION:") + `
  Supported filenames: gitv.json, giti.json, git-remote-color.json
  Lookup order: $GIT_REMOTE_COLOR_CONFIG → exe dir → cwd → config dir → home

` + color("#FEDE5D", "ENVIRONMENT:") + `
  GIT_REMOTE_COLOR_CONFIG    Path to custom config file
  GITHUB_TOKEN               Fallback GitHub token (config file takes priority)
  PAGER                      Custom pager binary (default: less)
`)
}

// ─── Default Config ───────────────────────────────────────────────────────────

func defaultConfig() Config {
	return Config{
		Remote:        "#00FFFF",
		Scheme:        "#FFAAFF",
		Host:          "#55AA00",
		Path:          "#AAAAFF",
		Repo:          "#FFFF00",
		Fetch:         "#00AAFF",
		Push:          "#AA5500",
		Description:   "#00AAFF",
		Branch:        "#FFAAFF",
		Tag:           "#AAAA00",
		Visibility:    "#00FFFF",
		LastUpdate:    "#FFFF00",
		ReadmeColor:   "#95E1D3",
		WorkflowColor: "#C3E88D",
		GlamourStyle:  "auto",
		GlamourWidth:  100,
	}
}

// ─── Parse Arguments ──────────────────────────────────────────────────────────

func parseArgs() Args {
	args := Args{Dir: "."}
	foundDir := false

	// Check for "config" command first
	if len(os.Args) > 1 && os.Args[1] == "config" {
		args.Command = "config"
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "show":
				args.Subcommand = "show"
				return args
			case "path":
				args.Subcommand = "path"
				return args
			default:
				args.Help = true
				return args
			}
		}
		args.Help = true
		return args
	}

	// Standard flag parsing
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			expanded := expandCombinedFlags(arg[1:])
			os.Args = append(os.Args[:i], append(expanded, os.Args[i+1:]...)...)
			i--
			continue
		}

		switch arg {
		case "-h", "--help":
			args.Help = true
			return args
		case "-v", "--version":
			args.Version = true
			return args
		case "-d", "--detail", "-r", "--readme":
			args.Detail = true
		case "-w", "--workflows":
			args.Workflows = true
		case "-f", "--full":
			args.FullOutput = true
		case "--no-cache":
			args.NoCache = true
		default:
			if !strings.HasPrefix(arg, "-") && !foundDir {
				args.Dir = arg
				foundDir = true
			}
		}
	}
	return args
}

// expandCombinedFlags turns "dfw" → ["-d", "-f", "-w"]
func expandCombinedFlags(flags string) []string {
	known := map[byte]string{
		'd': "-d", 'r': "-r", 'f': "-f",
		'w': "-w", 'h': "-h", 'v': "-v",
	}
	var out []string
	for i := 0; i < len(flags); i++ {
		if v, ok := known[flags[i]]; ok {
			out = append(out, v)
		}
	}
	return out
}

// ─── Load Config ──────────────────────────────────────────────────────────────

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func configCandidates() []string {
	exe, _ := os.Executable()
	// Resolve symlinks so we get the real binary location
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
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

	// fmt.Printf("cfg: %v\n", cfg)

	for _, path := range configCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		
		fmt.Printf("Config File: %s\n", path)

		// Merge: only override non-zero values so defaults survive partial configs
		var partial Config
		if err := json.Unmarshal(data, &partial); err == nil {
			mergeConfig(&cfg, partial)
		}
		break
	}

	if cfg.Token == "" {
			// Fallback: GITHUB_TOKEN env var (config file takes precedence)
			cfg.Token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}

	// Force environment variables to take absolute priority over config files
	if envToken := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); envToken != "" && cfg.Token == ""  {
	    cfg.Token = envToken
	} 

	return cfg
}

// mergeConfig copies non-zero fields from src into dst.
func mergeConfig(dst *Config, src Config) {
	if src.Remote != "" {
		dst.Remote = src.Remote
	}
	if src.Scheme != "" {
		dst.Scheme = src.Scheme
	}
	if src.Host != "" {
		dst.Host = src.Host
	}
	if src.Path != "" {
		dst.Path = src.Path
	}
	if src.Repo != "" {
		dst.Repo = src.Repo
	}
	if src.Fetch != "" {
		dst.Fetch = src.Fetch
	}
	if src.Push != "" {
		dst.Push = src.Push
	}
	if src.Description != "" {
		dst.Description = src.Description
	}
	if src.Branch != "" {
		dst.Branch = src.Branch
	}
	if src.Tag != "" {
		dst.Tag = src.Tag
	}
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.Visibility != "" {
		dst.Visibility = src.Visibility
	}
	if src.LastUpdate != "" {
		dst.LastUpdate = src.LastUpdate
	}
	if len(src.LanguageColors) > 0 {
		dst.LanguageColors = src.LanguageColors
	}
	if src.ReadmeColor != "" {
		dst.ReadmeColor = src.ReadmeColor
	}
	if src.WorkflowColor != "" {
		dst.WorkflowColor = src.WorkflowColor
	}
	if src.GlamourStyle != "" {
		dst.GlamourStyle = src.GlamourStyle
	}
	if src.GlamourWidth != 0 {
		dst.GlamourWidth = src.GlamourWidth
	}
	if src.Owner != "" {
		dst.Owner = src.Owner
	}
}

// ─── Repo Name Resolution ─────────────────────────────────────────────────────

// resolveRepoArg determines whether the argument is a path or a GitHub slug.
// It returns (user, repo, isSlug) when it looks like a GitHub reference.
//
// Resolution order:
//  1. Explicit "owner/repo" form — always treated as a slug.
//  2. Bare name that looks path-like (starts with . / ~ or contains a separator) — never a slug.
//  3. Bare name + cfg.Owner set — use config owner.
//  4. Bare name + no cfg.Owner — infer owner from remotes in the current git repo.
//  5. Bare name that doesn't exist as a local path — treat as slug with empty owner
//     (caller will error cleanly if owner is still unknown).
func resolveRepoArg(arg string, cfg Config) (user, repo string, isSlug bool) {
	// 1. Explicit owner/repo slug
	if parts := strings.SplitN(arg, "/", 2); len(parts) == 2 {
		u := strings.TrimSpace(parts[0])
		r := strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git")
		if isValidName(u) && isValidName(r) {
			return u, r, true
		}
	}

	// Anything path-like stays as a path
	if pathLike(arg) {
		return "", "", false
	}

	// Must be a valid repo name to continue
	if !isValidName(arg) {
		return "", "", false
	}

	// 2. If the name matches an existing local directory/file, treat as path
	// UNLESS owner is explicitly set in config (explicit config wins over filesystem)
	if cfg.Owner == "" {
		if fi, err := os.Stat(arg); err == nil && fi.IsDir() {
			return "", "", false
		}
	}

	// 3. Config owner takes priority
	if cfg.Owner != "" {
		return cfg.Owner, arg, true
	}

	// 4. Infer owner from remotes of the current git repo
	if owner := inferOwnerFromLocalRepo(); owner != "" {
		return owner, arg, true
	}

	// 5. No owner at all — still treat as slug; main() will print a helpful error
	return "", arg, true
}

// inferOwnerFromLocalRepo reads `git remote -v` from cwd and returns the GitHub
// user/org of the first github.com remote found.
func inferOwnerFromLocalRepo() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	root, err := findGitRoot(cwd)
	if err != nil {
		return ""
	}
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		r, ok := parse(line)
		if !ok || r.Host != "github.com" || r.User == "" {
			continue
		}
		return r.User
	}
	return ""
}

func isValidName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func pathLike(s string) bool {
	return strings.HasPrefix(s, ".") || strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "~") || strings.Contains(s, string(filepath.Separator))
}

// ─── Find Git Root ────────────────────────────────────────────────────────────

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
	return "", fmt.Errorf("not a git repository (or any parent up to mount point)")
}

// ─── URL Parser ───────────────────────────────────────────────────────────────

func parse(line string) (Row, bool) {
	f := strings.Fields(line)
	if len(f) < 2 {
		return Row{}, false
	}

	r := Row{Remote: f[0], URL: f[1]}
	if len(f) > 2 {
		r.Type = strings.Trim(f[2], "()")
	}

	url := r.URL

	// git@ SSH: git@github.com:user/repo.git
	if strings.HasPrefix(url, "git@") || (strings.Contains(url, "@") &&
		strings.Contains(url, ":") && !strings.Contains(url, "://")) {
		atIdx := strings.Index(url, "@")
		colonIdx := strings.Index(url[atIdx:], ":") + atIdx
		if atIdx < 0 || colonIdx <= atIdx {
			return r, false
		}
		r.Host = url[atIdx+1 : colonIdx]
		r.Scheme = "ssh://"
		rest := url[colonIdx+1:]
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			r.User = parts[0]
			r.Repo = strings.TrimSuffix(parts[1], ".git")
		} else if len(parts) == 1 {
			// e.g. git@github.com:user  (malformed but handle gracefully)
			r.User = parts[0]
		}
		return r, true
	}

	// HTTPS / HTTP / git://
	if idx := strings.Index(url, "://"); idx >= 0 {
		r.Scheme = url[:idx+3]
		rest := url[idx+3:]
		// Strip inline credentials: https://user:pass@host/...
		if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
			rest = rest[atIdx+1:]
		}
		parts := strings.SplitN(rest, "/", -1)
		if len(parts) >= 1 {
			r.Host = parts[0]
		}
		if len(parts) >= 2 {
			r.User = parts[1]
		}
		if len(parts) >= 3 {
			r.Repo = strings.TrimSuffix(parts[2], ".git")
		}
		if r.Host != "" && r.User != "" && r.Repo != "" {
			return r, true
		}
		// Still return partial result for non-GitHub hosts
		return r, r.Host != ""
	}

	return r, false
}

// ─── HTTP ─────────────────────────────────────────────────────────────────────

func getJSON(ctx context.Context, url, token string, target interface{}) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "git-remote-color/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if t := strings.TrimSpace(token); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // max 4 MB

	if resp.StatusCode != http.StatusOK {
		// Try to extract GitHub error message
		var ghErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &ghErr)
		return resp.StatusCode, HTTPError{Status: resp.StatusCode, Message: ghErr.Message}
	}

	if err := json.Unmarshal(body, target); err != nil {
		return resp.StatusCode, fmt.Errorf("decode JSON: %w", err)
	}
	return resp.StatusCode, nil
}

func formatDate(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatRelative(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// ─── README ───────────────────────────────────────────────────────────────────

func fetchReadme(ctx context.Context, user, repo, token string) *ReadmeInfo {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", user, repo)
	var readme ReadmeInfo
	if _, err := getJSON(ctx, url, token, &readme); err != nil {
		return nil
	}
	return &readme
}

func decodeBase64Content(s string) (string, error) {
	// GitHub wraps lines at 60 chars; strip whitespace before decode
	clean := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', ' ', '\t':
			return -1
		}
		return r
	}, s)
	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	return string(decoded), nil
}

// ─── Pager ────────────────────────────────────────────────────────────────────

func showInPager(content string) {
	if runtime.GOOS == "windows" {
		// Windows: use more.com, pager doesn't support -R
		cmd := exec.Command("more")
		cmd.Stdin = strings.NewReader(content)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Print(content)
		}
		return
	}

	pager := strings.TrimSpace(os.Getenv("PAGER"))
	if pager == "" {
		pager = "less"
	}

	pagerArgs := []string{"-R"}
	// Some pagers (like bat) don't accept -R
	if strings.Contains(pager, "bat") {
		pagerArgs = []string{"--paging=always", "--color=always"}
	}

	cmd := exec.Command(pager, pagerArgs...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Fallback chain: more → direct print
		cmd2 := exec.Command("more")
		cmd2.Stdin = strings.NewReader(content)
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		if err2 := cmd2.Run(); err2 != nil {
			fmt.Print(content)
		}
	}
}

// ─── Show README ──────────────────────────────────────────────────────────────

func showReadme(ctx context.Context, user, repo string, cfg Config, fullOutput bool) {
	fmt.Println("\n" + color("#FF6B6B", "═══ README ═══"))
	readme := fetchReadme(ctx, user, repo, cfg.Token)

	if readme == nil {
		fmt.Println("   " + color("#FFE66D", "⚠ No README found in this repository"))
		return
	}

	fmt.Println("   " + color(cfg.ReadmeColor, "📄 "+readme.Name))
	fmt.Println("   " + color("#888888", strings.Repeat("─", 60)))
	fmt.Println()

	var raw string
	switch readme.Encoding {
	case "base64":
		decoded, err := decodeBase64Content(readme.Content)
		if err != nil {
			fmt.Println("   " + color("#FF5555", "⚠ Could not decode README: "+err.Error()))
			return
		}
		raw = decoded
	case "":
		raw = readme.Content
	default:
		fmt.Println("   " + color("#FF5555", "⚠ Unknown README encoding: "+readme.Encoding))
		return
	}

	width := cfg.GlamourWidth
	if width <= 0 {
		width = 100
	}
	style := cfg.GlamourStyle
	if style == "" {
		style = "auto"
	}

	var renderer *glamour.TermRenderer
	var err error
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
		// Fallback: print raw without glamour
		fmt.Println("   " + color("#888888", "(glamour unavailable, showing raw)"))
		fmt.Println(raw)
		return
	}

	rendered, err := renderer.Render(raw)
	if err != nil {
		fmt.Println("   " + color("#FF5555", "⚠ Could not render README: "+err.Error()))
		return
	}

	lines := strings.Count(rendered, "\n")
	if fullOutput || lines <= 50 {
		fmt.Print(rendered)
	} else {
		fmt.Println(color("#888888", "   📖 Opening in pager (press q to quit, -f for direct output)"))
		showInPager(rendered)
	}
}

// ─── Workflow Status ──────────────────────────────────────────────────────────

// workflowIcon returns a colored icon + label for a workflow run.
func workflowIcon(run WorkflowRun) string {
	if run.Status == "in_progress" || run.Status == "queued" || run.Status == "waiting" {
		return color("#FFFF55", "⟳ "+run.Status)
	}
	switch run.Conclusion {
	case "success":
		return color("#55FF55", "✔ success")
	case "failure":
		return color("#FF5555", "✘ failure")
	case "cancelled":
		return color("#AAAAAA", "⊘ cancelled")
	case "skipped":
		return color("#888888", "⊖ skipped")
	case "timed_out":
		return color("#FF8800", "⏱ timed_out")
	case "action_required":
		return color("#FFB300", "⚠ action_required")
	case "neutral":
		return color("#AAAAFF", "◌ neutral")
	case "stale":
		return color("#888888", "↻ stale")
	default:
		return color("#AAAAAA", "? "+run.Conclusion)
	}
}

func showWorkflows(ctx context.Context, user, repo string, cfg Config) {
	fmt.Println("\n" + color("#FF6B6B", "═══ GitHub Actions ═══"))

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/runs?per_page=10",
		user, repo,
	)

	var runsResp WorkflowRunsResponse
	status, err := getJSON(ctx, url, cfg.Token, &runsResp)
	if err != nil {
		var httpErr HTTPError
		if isHTTPError(err, &httpErr) {
			switch httpErr.Status {
			case http.StatusNotFound:
				fmt.Println("   " + color("#FFE66D", "⚠ No workflows found (repo may have no Actions configured)"))
			case http.StatusForbidden:
				fmt.Println("   " + color("#FF5555", "🔒 Access denied — Actions may be disabled or token lacks permissions"))
			default:
				fmt.Printf("   %s (HTTP %d)\n", color("#FF5555", "❌ Failed to fetch workflows"), status)
			}
		} else {
			fmt.Println("   " + color("#888888", "⚠ Could not reach GitHub API: "+err.Error()))
		}
		return
	}

	if runsResp.TotalCount == 0 || len(runsResp.WorkflowRuns) == 0 {
		fmt.Println("   " + color("#FFE66D", "⚠ No workflow runs found"))
		return
	}

	fmt.Printf("   %s %d total run(s), showing last %d\n",
		color(cfg.WorkflowColor, "📋"),
		runsResp.TotalCount,
		min(len(runsResp.WorkflowRuns), 10),
	)
	fmt.Println()

	// Deduplicate: show latest run per workflow name
	seen := map[string]bool{}
	for _, run := range runsResp.WorkflowRuns {
		if seen[run.Name] {
			continue
		}
		seen[run.Name] = true

		branch := ""
		if run.HeadBranch != "" {
			branch = color("#FFAAFF", " ["+run.HeadBranch+"]")
		}
		event := ""
		if run.Event != "" {
			event = color("#888888", " ("+run.Event+")")
		}
		when := formatRelative(run.UpdatedAt)
		if when != "" {
			when = color("#888888", "  "+when)
		}

		fmt.Printf("   %s  %s%s%s%s\n",
			workflowIcon(run),
			color(cfg.WorkflowColor, run.Name),
			branch,
			event,
			when,
		)
	}
}

func isHTTPError(err error, target *HTTPError) bool {
	if he, ok := err.(HTTPError); ok {
		*target = he
		return true
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Fetch All ────────────────────────────────────────────────────────────────

func fetchAll(ctx context.Context, user, repo, token string, noCache bool) CacheEntry {
	key := user + "/" + repo

	if !noCache {
		cacheMu.RLock()
		if c, ok := cache[key]; ok && time.Since(time.Unix(c.Time, 0)) < cacheTTL {
			cacheMu.RUnlock()
			c.Cached = true
			return c
		}
		cacheMu.RUnlock()
	}

	entry := CacheEntry{}

	_, err := getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo, token, &entry.Repo)
	if err != nil {
		var httpErr HTTPError
		if isHTTPError(err, &httpErr) {
			msg := fmt.Sprintf("❌ error (HTTP %d)", httpErr.Status)
			switch httpErr.Status {
			case http.StatusNotFound:
				msg = "❌ repository not found"
			case http.StatusForbidden:
				msg = "🔒 access denied"
			case http.StatusUnauthorized:
				msg = "🔑 invalid or expired GitHub token"
			case http.StatusTooManyRequests: // 429
				msg = "⏳ API rate limit exceeded (add github_token for higher limits)"
			}
			// GitHub returns 403 for rate-limit exhaustion; detect via message body
			if strings.Contains(strings.ToLower(httpErr.Message), "rate limit") {
				msg = "⏳ API rate limit exceeded (add github_token for higher limits)"
			}
			return CacheEntry{Repo: GitHubRepo{Description: msg}}
		}

		// Network error: return cached stale data if any
		cacheMu.RLock()
		if c, ok := cache[key]; ok {
			cacheMu.RUnlock()
			c.Cached = true
			return c
		}
		cacheMu.RUnlock()
		return CacheEntry{Repo: GitHubRepo{Description: "⚠ offline (no cached data)"}}
	}

	// Parallel fetch of supplementary data
	type result struct {
		releases  []Release
		branches  []Branch
		tags      []Tag
		langRaw   LangMap
		workflows WorkflowRunsResponse
	}
	res := result{}
	var wg sync.WaitGroup
	var resMu sync.Mutex

	fetch := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	fetch(func() {
		var v []Release
		getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo+"/releases?per_page=100", token, &v)
		resMu.Lock()
		res.releases = v
		resMu.Unlock()
	})
	fetch(func() {
		var v []Branch
		getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo+"/branches?per_page=100", token, &v)
		resMu.Lock()
		res.branches = v
		resMu.Unlock()
	})
	fetch(func() {
		var v []Tag
		getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo+"/tags?per_page=100", token, &v)
		resMu.Lock()
		res.tags = v
		resMu.Unlock()
	})
	fetch(func() {
		var v LangMap
		getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo+"/languages", token, &v)
		resMu.Lock()
		res.langRaw = v
		resMu.Unlock()
	})
	fetch(func() {
		var v WorkflowRunsResponse
		getJSON(ctx, "https://api.github.com/repos/"+user+"/"+repo+"/actions/runs?per_page=10", token, &v)
		resMu.Lock()
		res.workflows = v
		resMu.Unlock()
	})

	wg.Wait()

	// Aggregate downloads
	total := 0
	for _, r := range res.releases {
		for _, a := range r.Assets {
			total += a.DownloadCount
		}
	}
	entry.Downloads = total

	for _, b := range res.branches {
		entry.Branches = append(entry.Branches, b.Name)
	}
	for _, t := range res.tags {
		entry.Tags = append(entry.Tags, t.Name)
	}
	entry.WorkflowRuns = res.workflows.WorkflowRuns

	// Language percentages
	langTotal := 0
	for _, v := range res.langRaw {
		langTotal += v
	}
	entry.Languages = map[string]float64{}
	if langTotal > 0 {
		for k, v := range res.langRaw {
			entry.Languages[k] = (float64(v) / float64(langTotal)) * 100
		}
	}

	entry.Time = time.Now().Unix()

	cacheMu.Lock()
	cache[key] = entry
	cacheMu.Unlock()

	return entry
}

// ─── Print Remote Info ────────────────────────────────────────────────────────

func printRemoteInfo(rows []Row, cfg Config, args Args) {
	r := rows[0]

	// Build colored URL line
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
		line += "  (" + strings.Join(types, ", ") + ")"
	}
	fmt.Println(line)

	if r.Host != "github.com" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data := fetchAll(ctx, r.User, r.Repo, cfg.Token, args.NoCache)
	printGitHubData(ctx, r.User, r.Repo, data, cfg, args)
}

// printGitHubData prints all GitHub metadata for a fetched CacheEntry.
// Used by both slug mode (gitv gntp) and local-repo mode (gitv).
func printGitHubData(ctx context.Context, user, repo string, data CacheEntry, cfg Config, args Args) {
	if data.Cached {
		fmt.Println("   " + color("#888888", "(cached)"))
	}

	desc := data.Repo.Description
	if isErrorSentinel(desc) {
		fmt.Println("   " + color("#FF5555", desc))
		return
	}
	if desc != "" {
		fmt.Println("   " + color(cfg.Description, desc))
	}

	visibility := color(cfg.Visibility, "🌍 public")
	if data.Repo.Private {
		visibility = color(cfg.Visibility, "🔒 private")
	}
	fmt.Printf("   %s  ⭐ %d  🍴 %d  🐞 %d  ⬇ %d  🕒 %s\n",
		visibility,
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
		langs := make([]langPair, 0, len(data.Languages))
		for k, v := range data.Languages {
			langs = append(langs, langPair{k, v})
		}
		sort.Slice(langs, func(i, j int) bool {
			return langs[i].Pct > langs[j].Pct
		})
		colors := getLangColors(cfg)
		parts := make([]string, 0, len(langs))
		for i, l := range langs {
			c := colors[i%len(colors)]
			parts = append(parts, color(c, fmt.Sprintf("%s %.1f%%", l.Name, l.Pct)))
		}
		fmt.Println("   🧠", strings.Join(parts, "  "))
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
		fmt.Println("   🏷️  tags:")
		for _, t := range data.Tags {
			fmt.Println("     -", color(cfg.Tag, t))
		}
	}

	// Workflows
	if args.Workflows || len(data.WorkflowRuns) > 0 {
		if args.Workflows {
			showWorkflows(ctx, user, repo, cfg)
		} else {
			printWorkflowSummary(data.WorkflowRuns, cfg)
		}
	}

	// README
	if args.Detail {
		showReadme(ctx, user, repo, cfg, args.FullOutput)
	}
}

// printWorkflowSummary shows a compact one-liner when -w is not used.
func printWorkflowSummary(runs []WorkflowRun, cfg Config) {
	if len(runs) == 0 {
		return
	}
	latest := runs[0]
	fmt.Printf("   %s %s\n",
		color(cfg.WorkflowColor, "⚙ CI:"),
		workflowIcon(latest)+" "+color("#888888", latest.Name),
	)
}

func isErrorSentinel(s string) bool {
	return strings.HasPrefix(s, "❌") ||
		strings.HasPrefix(s, "🔒") ||
		strings.HasPrefix(s, "🔑") ||
		strings.HasPrefix(s, "⚠") ||
		strings.HasPrefix(s, "⏳")
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	args := parseArgs()

	if args.Help {
		printHelp()
		return
	}
	if args.Version {
		printVersion()
		return
	}

	cfg := loadConfig()

	// Route "config show" and "config path"
	if args.Command == "config" {
		if args.Subcommand == "path" {
			fmt.Println(color("#4ECDC4", "🔎 Configuration search paths (ordered by priority):"))
			candidates := configCandidates()
			for _, path := range candidates {
				if _, err := os.Stat(path); err == nil {
					fmt.Printf("  %s %s\n", color("#55FF55", "✔ [ACTIVE]"), path)
				} else {
					fmt.Printf("             %s\n", color("#888888", path))
				}
			}
			return
		}

		if args.Subcommand == "show" {
			pretty, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				fmt.Println(color("#FF5555", "❌ Error formatting configuration:"), err)
				os.Exit(1)
			}
			fmt.Println(color("#FFAAFF", "📝 Active Configuration State:"))
			fmt.Println(string(pretty))
			return
		}
	}

	// Check if the argument looks like a GitHub slug before treating it as a path
	if user, repo, ok := resolveRepoArg(args.Dir, cfg); ok {
		// No owner could be determined — give a clear, actionable error
		if user == "" {
			fmt.Println(color("#FF5555", "❌ Cannot resolve repo name: owner is unknown"))
			fmt.Println(color("#888888", "   Fix one of:"))
			fmt.Println(color("#888888", `   • Set "owner": "yourname" in gitv.json`))
			fmt.Println(color("#888888", "   • Use the full form: git-remote-color owner/"+repo))
			fmt.Println(color("#888888", "   • Run from inside a GitHub-backed git repository"))
			os.Exit(1)
		}

		// Direct GitHub slug mode — show info without a local repo
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fmt.Printf("%s %s/%s\n",
			color("#888888", "🔎 Fetching info for"),
			color(cfg.Path, user),
			color(cfg.Repo, repo),
		)

		data := fetchAll(ctx, user, repo, cfg.Token, args.NoCache)
		printGitHubData(ctx, user, repo, data, cfg, args)
		return
	}

	// Path mode — find a local git repo
	dir := args.Dir
	if strings.HasPrefix(dir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, dir[1:])
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println(color("#FF5555", "❌ Error:"), "Invalid path:", dir)
		os.Exit(1)
	}

	root, err := findGitRoot(absDir)
	if err != nil {
		fmt.Println(color("#FF5555", "❌"), "Not a git repository:", absDir)
		fmt.Println(color("#888888", "   Tip: run 'git init' or navigate to a git repository"))
		os.Exit(1)
	}

	if dir != "." && dir != absDir {
		fmt.Println(color("#888888", "📂 Repository:"), color(cfg.Path, root))
	}

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = root

	out, err := cmd.Output()
	if err != nil {
		fmt.Println(color("#FF5555", "❌ git remote -v failed:"), err)
		os.Exit(1)
	}

	// Group rows by URL so fetch/push pairs appear together
	type urlGroup struct {
		rows []Row
		// preserve insertion order
	}
	order := []string{}
	group := map[string][]Row{}

	for _, l := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		r, ok := parse(l)
		if !ok {
			continue
		}
		if _, exists := group[r.URL]; !exists {
			order = append(order, r.URL)
		}
		group[r.URL] = append(group[r.URL], r)
	}

	if len(group) == 0 {
		fmt.Println(color("#FFE66D", "⚠ No remote repositories configured"))
		fmt.Println(color("#888888", "   Add a remote with: git remote add origin <url>"))
		return
	}

	for _, url := range order {
		printRemoteInfo(group[url], cfg, args)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"sort"
)

type LangMap map[string]int

type Config struct {
	Remote      string `json:"remote"`
	Scheme      string `json:"scheme"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Repo        string `json:"repo"`
	Fetch       string `json:"fetch"`
	Push        string `json:"push"`
	Description string `json:"description"`
	Branch      string `json:"branch"`
	Tag         string `json:"tag"`
	Token       string `json:"github_token"`
	Visibility string `json:"visibility"`
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
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	Issues      int    `json:"open_issues_count"`
	Private     bool   `json:"private"`
}

type Branch struct{ Name string `json:"name"` }
type Tag struct{ Name string `json:"name"` }

type CacheEntry struct {
	Repo      GitHubRepo
	Branches  []string
	Tags      []string
	Languages map[string]float64
	Time      int64
	Cached    bool
}

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	cache      = map[string]CacheEntry{}
	mu         sync.Mutex
)

// ---------- COLOR ----------
func color(hex, text string) string {
	if hex == "" {
		return text
	}
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// ---------- DEFAULT CONFIG ----------
func defaultConfig() Config {
	return Config{
		Remote: "#00FFFF", Scheme: "#FFAAFF", Host: "#55AA00",
		Path: "#AAAAFF", Repo: "#FFFF00",
		Fetch: "#00AAFF", Push: "#AA5500",
		Description: "#00AAFF",
		Branch: "#FFAAFF", Tag: "#AAAA00",
		Visibility: "#00FFFF",
	}
}

// ---------- LOAD CONFIG ----------
func loadConfig() Config {
	cfg := defaultConfig()

	exe, _ := os.Executable()
	path := strings.TrimSuffix(exe, filepath.Ext(exe)) + ".json"

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	_ = json.Unmarshal(data, &cfg)
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

// ---------- HTTP ----------
func getJSON(url, token string, target interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "git-remote-color")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	// retry with token if rate limit
	if resp.Header.Get("X-RateLimit-Remaining") == "0" && token != "" {
		req2, _ := http.NewRequest("GET", url, nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		resp2, _ := httpClient.Do(req2)
		defer resp2.Body.Close()

		body, _ := io.ReadAll(resp2.Body)
		return json.Unmarshal(body, target)
	}

	body, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(body, target)
}

// ---------- FETCH ----------
func fetchAll(user, repo, token string) CacheEntry {
	key := user + "/" + repo

	mu.Lock()
	if c, ok := cache[key]; ok && time.Now().Unix()-c.Time < 3600 {
		mu.Unlock()
		// return c
		c.Cached = true
		return c
	}
	mu.Unlock()

	entry := CacheEntry{}

	err := getJSON("https://api.github.com/repos/"+user+"/"+repo, token, &entry.Repo)

	if err != nil {
		// 🔥 fallback to cache
		mu.Lock()
		if c, ok := cache[key]; ok {
			mu.Unlock()
			// return c
			c.Cached = true
			return c
		}
		mu.Unlock()

		// ❌ no cache + no internet
		return CacheEntry{
			Repo: GitHubRepo{
				Description: "⚠ offline (no cache available)",
			},
		}
	}

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

	// total := 0
	// for _, v := range langRaw {
	// 	total += v
	// }

	// entry.Languages = map[string]float64{}

	// for k, v := range langRaw {
	// 	entry.Languages[k] = (float64(v) / float64(total)) * 100
	// }

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
	cfg := loadConfig()

	// -------- PATH SUPPORT (FIXED) --------
	dir := "."
	for i, a := range os.Args {
		if i == 0 {
			continue
		}
		if !strings.HasPrefix(a, "-") {
			dir = a
			break
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println("Invalid path:", dir)
		return
	}

	root, err := findGitRoot(absDir)
	if err != nil {
		fmt.Println("Not a git repository:", absDir)
		return
	}

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = root // 🔥 THIS FIXES YOUR BUG

	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
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

			// ✅ cached indicator (NOW valid)
			if data.Cached {
				fmt.Println("   " + color("#888888", "(cached)"))
			}

			// ✅ offline handling
			if data.Repo.Description == "⚠ offline (no cache available)" {
				fmt.Println("   " + color("#FF5555", "⚠ offline (no cached data)"))
				continue
			}

			if data.Repo.Description != "" {
				fmt.Println("   " + color(cfg.Description, data.Repo.Description))
			}

			visibility := "🌍 public"
			if data.Repo.Private {
				visibility = "🔒 private"
			}

			fmt.Printf("   %s  ⭐ %d  🍴 %d  🐞 %d\n",
				color(cfg.Visibility, visibility),
				data.Repo.Stars,
				data.Repo.Forks,
				data.Repo.Issues,
			)

			// languages
			// if len(data.Languages) > 0 {
			// 	var parts []string
			// 	for lang, pct := range data.Languages {
			// 		parts = append(parts, fmt.Sprintf("%s %.1f%%", lang, pct))
			// 	}
			// 	fmt.Println("   🧠", strings.Join(parts, ", "))
			// }

			if len(data.Languages) > 0 {
				type langPair struct {
					Name string
					Pct  float64
				}

				var langs []langPair
				for k, v := range data.Languages {
					langs = append(langs, langPair{k, v})
				}

				// 🔥 sort descending
				sort.Slice(langs, func(i, j int) bool {
					return langs[i].Pct > langs[j].Pct
				})

				var parts []string
				for _, l := range langs {
					parts = append(parts, fmt.Sprintf("%s %.1f%%", l.Name, l.Pct))
				}

				fmt.Println("   🧠", strings.Join(parts, ", "))
			}

			// branches
			if len(data.Branches) > 0 {
				fmt.Println("   🌿 branches:")
				for _, b := range data.Branches {
					fmt.Println("     -", color(cfg.Branch, b))
				}
			}

			// tags
			if len(data.Tags) > 0 {
				fmt.Println("   🏷️ tags:")
				for _, t := range data.Tags {
					fmt.Println("     -", color(cfg.Tag, t))
				}
			}
		}
	}
}
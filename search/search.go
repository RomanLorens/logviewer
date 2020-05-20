package search

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RomanLorens/logviewer/config"
	e "github.com/RomanLorens/logviewer/error"
	log "github.com/RomanLorens/logviewer/logger"
)

//Application application
type Application struct {
	ApplicationID string `json:"application"`
	Env           string `json:"env"`
}

//Search search
type Search struct {
	Value    string `json:"value"`
	FromTime int64  `json:"FromTime"`
	ToTime   int64  `json:"ToTime"`
	Application
}

//Result search result
type Result struct {
	LogFile string   `json:"logfile"`
	Lines   []string `json:"lines"`
	Host    string   `json:"host"`
	Error   *e.Error `json:"error,omitempty"`
	Time    int64    `json:"time"`
}

//Find find logs
func (s *Search) Find(ctx context.Context) ([]*Result, *e.Error) {
	log.Info(ctx, "Find %v", s)
	cfg, err := s.validate()
	if err != nil {
		return nil, err
	}
	hostname, er := os.Hostname()
	if er != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not obtain os hostname, %v", er)
	}
	out := make(chan []*Result, len(cfg.Hosts))
	for _, host := range cfg.Hosts {
		go func(host config.Host) {
			log.Info(ctx, "starting goroutine for %v", host.Endpoint)
			start := time.Now()
			local := isLocal(host.Endpoint, hostname)
			var res []*Result
			if local {
				res = localSearch(ctx, host, s)
			} else {
				//res = remoteSearch(ctx, host.Endpoint, host, s)
			}
			end := time.Now()
			elapsed := end.Sub(start)
			for _, r := range res {
				r.Time = elapsed.Milliseconds()
			}
			log.Info(ctx, "goroutine for %v finished", host.Endpoint)
			out <- res
		}(host)
	}
	return <-out, nil
}

//LogDetails log details
type LogDetails struct {
	ModTime int64  `json:"modtime"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
}

//ListLogs list logs for app
func ListLogs(ctx context.Context, app *Application) ([]*LogDetails, *e.Error) {
	logs := make([]*LogDetails, 0)
	hostname, er := os.Hostname()
	if er != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not obtain os hostname, %v", er)
	}
	for _, c := range config.ApplicationsConfig {
		if !(app.ApplicationID == c.Application && app.Env == c.Env) {
			continue
		}
		for _, h := range c.Hosts {
			local := isLocal(h.Endpoint, hostname)
			if local {
				for _, p := range h.Paths {
					dir := filepath.Dir(p)
					filename := filepath.Base(p)
					l, err := getStats(dir, filename)
					if err != nil {
						log.Error(ctx, err.Message)
						continue
					}
					logs = append(logs, l...)
				}
			} else {
				log.Error(ctx, "remote list logs not implemented")
			}
		}
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].ModTime > logs[j].ModTime })
	return logs, nil
}

func getStats(dir string, filename string) ([]*LogDetails, *e.Error) {
	logs := make([]*LogDetails, 0)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, e.Errorf(http.StatusBadRequest, err.Error())
	}
	if !info.IsDir() {
		return nil, e.Errorf(http.StatusBadRequest, "path %v is not dir", dir)
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		info, err = os.Stat(path)
		if err == nil && !info.IsDir() {
			logs = append(logs, &LogDetails{
				ModTime: info.ModTime().Unix(),
				Name:    info.Name(),
				Size:    info.Size(),
			})
		}
		return nil
	})
	return logs, nil
}

func localSearch(ctx context.Context, host config.Host, s *Search) []*Result {
	log.Info(ctx, "Local search for %v", host)
	out := make([]*Result, 0, len(host.Paths))
	for _, p := range host.Paths {
		r := Result{LogFile: p, Host: host.Endpoint}
		lines, err := grepFile(p, s)
		r.Error = err
		r.Lines = lines
		out = append(out, &r)
	}
	return out
}

func grepFile(path string, s *Search) ([]string, *e.Error) {
	out := make([]string, 0, 20)
	f, err := os.Open(path)
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not open file %v, %v", path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	val := strings.ToLower(s.Value)
	for scanner.Scan() {
		if strings.Contains(strings.ToLower(scanner.Text()), val) {
			out = append(out, normalizeText(scanner.Text()))
		}
	}
	var er *e.Error
	if err = scanner.Err(); err != nil {
		er = e.Errorf(http.StatusInternalServerError, "error when grepping file %v", err)
	}
	return out, er
}

func normalizeText(t string) string {
	t = strings.ReplaceAll(t, "\033[0;31mERROR\033[0m", "ERROR")
	t = strings.ReplaceAll(t, "\033[0;33mWARNING\033[0m", "WARNING")
	t = strings.ReplaceAll(t, "\033[0;32mINFO\033[0m", "INFO")
	return t
}

func remoteSearch(ctx context.Context, url string, host config.Host, s *Search) []*Result {
	log.Info(ctx, "Remote search for %v", host.Paths)
	r := make([]*Result, len(host.Paths))
	return r
}

func isLocal(endpoint string, hostname string) bool {
	if strings.Contains(strings.ToLower(endpoint), strings.ToLower(hostname)) ||
		strings.Contains(endpoint, "://localhost") {
		return true
	}
	return false
}

func (s *Search) validate() (*config.Config, *e.Error) {
	if strings.TrimSpace(s.Value) == "" {
		return nil, e.Errorf(http.StatusBadRequest, "Missing value")
	}
	if strings.TrimSpace(s.ApplicationID) == "" {
		return nil, e.Errorf(http.StatusBadRequest, "Missing application id")
	}
	if strings.TrimSpace(s.Env) == "" {
		return nil, e.Errorf(http.StatusBadRequest, "Missing environment")
	}
	var cfg *config.Config
	for _, c := range config.ApplicationsConfig {
		if c.Application == s.ApplicationID && c.Env == s.Env {
			cfg = c
			break
		}
	}
	if cfg == nil {
		return nil, e.Errorf(http.StatusBadRequest, "Could not find cfg for %v and %v host", s.ApplicationID, s.Env)
	}
	return cfg, nil
}

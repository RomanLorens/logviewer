package stat

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/RomanLorens/logviewer/config"
	e "github.com/RomanLorens/logviewer/error"
	"github.com/RomanLorens/logviewer/logger"
	"github.com/RomanLorens/logviewer/search"
)

//ReqID req id
type ReqID struct {
	ReqID string `json:"reqid"`
	Data  string `json:"date"`
}

//Stat stats
type Stat struct {
	LastTime string         `json:"lastTime"`
	Counter  int            `json:"counter"`
	Levels   map[string]int `json:"levels"`
	Errors   []*ReqID       `json:"errors"`
	Warnings []*ReqID       `json:"warnings"`
}

//Get gets stats
func Get(ctx context.Context, app *search.Application) (map[string]*Stat, *e.Error) {
	var cfg *config.LogStructure
	ok := false
	for _, c := range config.ApplicationsConfig {
		if c.Application == app.ApplicationID && c.Env == app.Env {
			cfg = c.LogStructure
			ok = true
			break
		}
	}
	if !ok {
		return nil, e.Errorf(500, "Missing logstructure confgig for %v %v", app.ApplicationID, app.Env)
	}

	if search.IsLocal(ctx, app.Host) {
		logger.Info(ctx, "Checking locally stats")
		return stats(app.Log, cfg)
	}
	return remoteStats(ctx, app)

}

func remoteStats(ctx context.Context, app *search.Application) (map[string]*Stat, *e.Error) {
	logger.Info(ctx, "Stats log remotely")
	var res map[string]*Stat
	url := search.ApiURL(app.Host, "stats")
	body, err := search.CallAPI(ctx, url, app)
	if err != nil {
		return nil, err
	}
	if er := json.Unmarshal(body, &res); er != nil {
		return nil, e.Errorf(500, "Could not read unmarshal, %v", er)
	}
	return res, nil
}

func stats(log string, cfg *config.LogStructure) (map[string]*Stat, *e.Error) {
	out := make(map[string]*Stat, 0)
	requests := make(map[string]int, 0)
	file, err := os.Open(log)
	if err != nil {
		return nil, e.Errorf(500, "Could not open log file, %v", err)
	}
	defer file.Close()

	maxTokens := max(cfg)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tokens := strings.Split(scanner.Text(), "|")
		if len(tokens) < maxTokens {
			continue
		}
		user := tokens[cfg.User]
		if len(strings.TrimSpace(user)) == 0 {
			continue
		}
		u, ok := out[user]
		if !ok {
			u = &Stat{
				Levels: make(map[string]int, 0),
			}
			out[user] = u
		}
		level := search.NormalizeText(tokens[cfg.Level])
		requests[tokens[cfg.Reqid]+level]++
		if requests[tokens[cfg.Reqid]+level] > 1 {
			continue
		}
		u.LastTime = tokens[cfg.Date]
		u.Counter++
		u.Levels[level]++
		if strings.ToUpper(level) == "ERROR" {
			u.Errors = append(u.Errors, &ReqID{
				tokens[cfg.Reqid], tokens[cfg.Date],
			})
		}
		if strings.ToUpper(level) == "WARNING" {
			u.Warnings = append(u.Warnings, &ReqID{
				tokens[cfg.Reqid], tokens[cfg.Date],
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, e.Errorf(500, "Error from scanner, %v", err)
	}
	for _, v := range out {
		for i, j := 0, len(v.Errors)-1; i < j; i, j = i+1, j-1 {
			v.Errors[i], v.Errors[j] = v.Errors[j], v.Errors[i]
		}
		for i, j := 0, len(v.Warnings)-1; i < j; i, j = i+1, j-1 {
			v.Warnings[i], v.Warnings[j] = v.Warnings[j], v.Warnings[i]
		}
	}
	return out, nil
}

func max(cfg *config.LogStructure) int {
	m := cfg.Date
	if cfg.User > m {
		m = cfg.User
	}
	if cfg.Reqid > m {
		m = cfg.Reqid
	}
	if cfg.Level > m {
		m = cfg.Level
	}
	return m
}

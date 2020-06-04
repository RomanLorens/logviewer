package search

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	e "github.com/RomanLorens/logviewer/error"
	log "github.com/RomanLorens/logviewer/logger"
)

//Application application
type Application struct {
	ApplicationID string `json:"application"`
	Env           string `json:"env"`
	Log           string `json:"log"`
	Host          string `json:"host"`
}

//Search search
type Search struct {
	Value         string   `json:"value"`
	FromTime      int64    `json:"FromTime"`
	ToTime        int64    `json:"ToTime"`
	ApplicationID string   `json:"application"`
	Env           string   `json:"env"`
	Logs          []string `json:"logs"`
	Hosts         []string `json:"hosts"`
}

//Result search result
type Result struct {
	LogFile string   `json:"logfile"`
	Lines   []string `json:"lines"`
	Host    string   `json:"host"`
	Error   *e.Error `json:"error,omitempty"`
	Time    int64    `json:"time"`
}

//LogDetails log details
type LogDetails struct {
	ModTime int64  `json:"modtime"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Host    string `json:"host"`
}

//LogSearch available log actions
type LogSearch interface {
	Tail(ctx context.Context, app *Application) (*Result, *e.Error)
	Grep(ctx context.Context, host string, s *Search) []*Result
	List(ctx context.Context, url string, s *Search) ([]*LogDetails, *e.Error)
}

var (
	tailSizeKB = 16
	ls         = LocalSearch{}
	rs         = RemoteSearch{}
)

//TailLog tail log
func TailLog(ctx context.Context, app *Application) (*Result, *e.Error) {
	if isLocal(ctx, app.Host) {
		return ls.Tail(ctx, app)
	}
	return rs.Tail(ctx, app)
}

//Find find logs
func Find(ctx context.Context, s *Search) ([]*Result, *e.Error) {
	log.Info(ctx, "Find %v", s)
	if err := validate(s); err != nil {
		return nil, err
	}
	out := make(chan []*Result, len(s.Hosts))
	for _, host := range s.Hosts {
		go func(host string) {
			log.Info(ctx, "starting goroutine for %v", host)
			start := time.Now()
			local := isLocal(ctx, host)
			var res []*Result
			if local {
				res = ls.Grep(ctx, host, s)
			} else {
				r, err := rs.Grep(ctx, host, s)
				if err != nil {
					res = append(res, &Result{Error: err, Time: 0})
				} else {
					res = append(res, r...)
				}
			}
			end := time.Now()
			elapsed := end.Sub(start)
			for _, r := range res {
				r.Time = elapsed.Milliseconds()
			}
			log.Info(ctx, "goroutine for %v finished", host)
			out <- res
		}(host)
	}
	return <-out, nil
}

//ListLogs list logs for app
func ListLogs(ctx context.Context, s *Search) ([]*LogDetails, *e.Error) {
	logs := make([]*LogDetails, 0)
	hc := make(chan string, len(s.Hosts))
	for _, h := range s.Hosts {
		go func(host string) {
			log.Info(ctx, "host routine for %v started...", host)
			if isLocal(ctx, host) {
				l, _ := ls.List(ctx, host, s) //no error for local
				logs = append(logs, l...)
			} else {
				l, err := rs.List(ctx, host, s)
				if err == nil {
					logs = append(logs, l...)
				} else {
					log.Error(ctx, "Error from api, %v", err)
				}
			}
			hc <- host
			log.Info(ctx, "host routine for %v finsihed", host)
		}(h)
	}
	<-hc //wait for all threads
	sort.Slice(logs, func(i, j int) bool { return logs[i].ModTime > logs[j].ModTime })
	return logs, nil
}

func isLocal(ctx context.Context, host string) bool {
	hostname, err := os.Hostname()
	if err != nil {
		log.Error(ctx, "Could not check hostname, %v", err)
		return false
	}
	if strings.Contains(strings.ToLower(host), strings.ToLower(hostname)) ||
		strings.Contains(host, "://localhost") {
		return true
	}
	return false
}

func validate(s *Search) *e.Error {
	if strings.TrimSpace(s.Value) == "" {
		return e.Errorf(http.StatusBadRequest, "Missing value")
	}
	if len(s.Hosts) == 0 {
		return e.Errorf(http.StatusBadRequest, "Missing hosts")
	}
	if len(s.Logs) == 0 {
		return e.Errorf(http.StatusBadRequest, "Missing logs")
	}
	return nil
}

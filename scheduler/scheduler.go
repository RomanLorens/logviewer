package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RomanLorens/logviewer-module/model"
	s "github.com/RomanLorens/logviewer-module/scheduler"
	"github.com/RomanLorens/logviewer-module/utils"
	"github.com/RomanLorens/logviewer/common"
	"github.com/RomanLorens/logviewer/config"
	l "github.com/RomanLorens/logviewer/logger"
	"github.com/RomanLorens/logviewer/mail"
	"github.com/RomanLorens/logviewer/resolver"
	"github.com/RomanLorens/logviewer/template"
)

var (
	logger     = l.L
	dateFormat = "2006-01-02"
	email      = mail.NewEmail(config.Config.EmailServer)
)

//InitScheduler starts scheduler
func InitScheduler() {

	scheduler := s.NewScheduler(logger)

	task := s.Task{Name: "stats collector",
		Run: func(ctx context.Context) {
			yesterday := time.Now().AddDate(0, 0, -1)
			yd := yesterday.Format(dateFormat)
			stats, err := PopulateStatsBatch(ctx, yd)
			if err == nil && len(stats) > 0 {
				msg, err := template.StatsTemplate(stats, "template/stats.html")
				if err != nil {
					logger.Error(ctx, "Could not parse template, %v", err)
					return
				}
				err = email.Send(ctx, config.Config.StatsEmailReciepients, fmt.Sprintf("Application Stats %v", yd), msg)
				if err != nil {
					logger.Error(ctx, "stats scheduler email error, %v", err)
				}
			}
		},
	}

	scheduler.Schedule(context.Background(), &task, time.Hour*4)
}

//PopulateStatsBatch collect and save per date
func PopulateStatsBatch(ctx context.Context, date string) ([]common.Stats, error) {
	t, er := time.Parse(dateFormat, date)
	if er != nil {
		return nil, fmt.Errorf("could not parse date, %v", er)
	}
	today := time.Now().Format(dateFormat)
	if today == date {
		return nil, fmt.Errorf("Can not collect stats for today")
	}

	dateStats, err := config.GetStatsKeys(ctx, date)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	var mutex = &sync.Mutex{}
	out := make([]common.Stats, 0, 10)
	headers := make(map[string][]string, 1)
	headers["Authorization"] = []string{fmt.Sprintf("Bearer %v", config.Config.Bearer)}
	//todo remove config dependency
	for _, app := range config.Config.ApplicationsConfig {
		if !app.CollectStats {
			continue
		}
		_date := t.Format(app.LogStructure.DateFormat)
		for _, host := range app.Hosts {
			for _, path := range host.Paths {
				key := common.StatsKey(&common.Stats{App: app.Application, Env: app.Env, LogPath: path, Date: date})
				if _, ok := dateStats[key]; ok {
					logger.Info(ctx, "stats per '%v' key already in db", key)
					continue
				}
				wg.Add(1)
				go func(path string, host config.Host, app config.AppConfig, date string, key string) {
					utils.CatchError(ctx, logger)
					defer wg.Done()
					logger.Info(ctx, "scheduler thread for get stats per '%v' key...", key)
					csr := common.CollectStatsRequest{StatsRequest: &common.StatsRequest{LogViewerEndpoint: host.Endpoint,
						StatsRequest: &model.StatsRequest{Log: path, LogStructure: app.LogStructure}}, Date: date}
					s, err := resolver.CollectStats(context.Background(), &csr, headers)
					if err != nil {
						logger.Error(ctx, err.Error())
						return
					}
					d := strings.ReplaceAll(date, "/", "-")
					stats := common.Stats{Stats: s, LogPath: path, App: app.Application, Env: app.Env, Date: d}
					err = config.SaveStats(ctx, &stats)
					if err != nil {
						logger.Error(ctx, "Error when saving stats, %v", err)
						return
					}
					mutex.Lock()
					defer mutex.Unlock()
					out = append(out, stats)
				}(path, host, app, _date, key)
			}
		}
	}
	wg.Wait()
	return out, nil
}

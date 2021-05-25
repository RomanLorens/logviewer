package template

import (
	"fmt"
	"sync"
	"testing"

	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/common"
)

func TestStatsTemplate(t *testing.T) {
	stats := make([]common.Stats, 0)
	users := make(map[string]map[string]int, 1)
	users["rl78794"] = make(map[string]int)
	users["rl78794"]["INFO"] = 12
	users["us12345"] = make(map[string]int)
	users["us12345"]["INFO"] = 55
	users["us12345"]["ERROR"] = 7
	stats = append(stats, common.Stats{App: "Training", LogPath: "/app/out.log", Env: "sit", Date: "2021-01-01", Stats: &model.CollectStatsRsults{TotalRequests: 25, Users: users}})
	stats = append(stats, common.Stats{App: "Deployment", LogPath: "/app/deployment.log", Env: "sit", Date: "2021-01-01", Stats: &model.CollectStatsRsults{TotalRequests: 9}})
	stats = append(stats, common.Stats{App: "Training", LogPath: "/app/out.log", Env: "uat", Date: "2021-01-01", Stats: &model.CollectStatsRsults{TotalRequests: 17}})

	res, err := StatsTemplate(stats, "stats.html")
	fmt.Println(res)
	if err != nil {
		t.Errorf("Failed %v", err)
	}
}

func TestRoutines(t *testing.T) {
	var wg sync.WaitGroup
	var mutex = &sync.Mutex{}
	out := make([]int, 0, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			mutex.Lock()
			defer mutex.Unlock()
			out = append(out, i)
		}(i)
	}
	wg.Wait()

	t.Logf("res = %v", out)
	if len(out) != 5 {
		t.Errorf("Failed %v", out)
	}
}

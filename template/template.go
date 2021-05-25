package template

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/RomanLorens/logviewer/common"
)

//StatsTemplate stats template
func StatsTemplate(data []common.Stats, path string) (string, error) {
	t, err := template.ParseFiles(path)
	if err != nil {
		return "", err
	}

	_data := make([]common.StatsTemplate, 0, len(data))
	for i, s := range data {
		errors := 0
		ur := make([]common.UserTotalRequests, 0, len(s.Stats.Users))
		for user, userLevels := range s.Stats.Users {
			totalReq := 0
			for level, count := range userLevels {
				if strings.Contains(level, "ERROR") || strings.Contains(level, "WARN") {
					errors += count
				}
				totalReq += count
			}
			ur = append(ur, common.UserTotalRequests{User: user, Count: totalReq, Level: userLevels})
		}
		sort.Slice(ur, func(i, j int) bool {
			return ur[i].Count > ur[j].Count
		})
		_data = append(_data, common.StatsTemplate{Stats: s, TotalErrors: errors, Users: ur})
		_data[i].LogPath = filepath.Base(s.LogPath)
	}

	sort.Slice(_data, func(i, j int) bool {
		return _data[i].App+_data[i].Env < _data[j].App+_data[j].Env
	})

	buf := new(bytes.Buffer)
	err = t.Execute(buf, _data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

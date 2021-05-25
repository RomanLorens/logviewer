package common

import (
	"fmt"

	"github.com/RomanLorens/logviewer-module/model"
)

//Stats save stats
type Stats struct {
	Stats     *model.CollectStatsRsults `json:"stats" bson:"stats"`
	LogPath   string                    `json:"logPath" bson:"logPath"`
	Date      string                    `json:"date"`
	App       string                    `json:"app"`
	Env       string                    `json:"env"`
	CreatedOn string                    `json:"createdOn" bson:"createdOn"`
}

//StatsTemplate stats template
type StatsTemplate struct {
	Stats
	TotalErrors int
	Users       []UserTotalRequests
}

//UserTotalRequests user requests count
type UserTotalRequests struct {
	User  string
	Count int
	Level map[string]int
}

//CollectStatsRequest req
type CollectStatsRequest struct {
	*StatsRequest
	Date string `json:"date"`
}

//AppCollectStatsRequest AppCollectStatsRequest
type AppCollectStatsRequest struct {
	*CollectStatsRequest
	App string `json:"app"`
	Env string `json:"env"`
	Log string `json:"log"`
}

//StatsRequest stats request
type StatsRequest struct {
	*model.StatsRequest
	LogViewerEndpoint string `json:"endpoint"`
}

//ErrorsRequest errors request
type ErrorsRequest struct {
	*model.StatsRequest
	LogViewerEndpoint string `json:"endpoint"`
	From              int    `json:"from"`
	Size              int    `json:"size"`
}

//StatReq stats req
type StatReq struct {
	App  string `json:"app"`
	Env  string `json:"env"`
	Log  string `json:"log"`
	From int64  `json:"from"`
	To   int64  `json:"to"`
}

//HostDetails host details
type HostDetails struct {
	LogViewerEndpoint string   `json:"endpoint"`
	Logs              []string `json:"paths"`
}

//SearchRequest search request
type SearchRequest struct {
	Hosts []HostDetails `json:"hosts"`
	Value string        `json:"value"`
}

//TailLogRequest tail log request
type TailLogRequest struct {
	LogViewerEndpoint string `json:"endpoint"`
	Log               string `json:"log"`
}

//StatsKey stats key
func StatsKey(s *Stats) string {
	return fmt.Sprintf("%v#%v#%v#%v", s.App, s.Env, s.Date, s.LogPath)
}

package resolver

import (
	"testing"

	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/common"
)

func TestRemoteSearch(t *testing.T) {
	hosts := make([]common.HostDetails, 0, 1)
	hosts = append(hosts, common.HostDetails{LogViewerEndpoint: remoteLVMEndpoint, Logs: []string{remoteLog}})
	sr := common.SearchRequest{Hosts: hosts, Value: "bc23456"}
	search(t, &sr)
}

func TestRemoteListLogs(t *testing.T) {
	host := common.HostDetails{LogViewerEndpoint: remoteLVMEndpoint, Logs: []string{remoteLog}}
	listLogs(t, &host)
}

func TestRemoteTailLog(t *testing.T) {
	req := common.TailLogRequest{LogViewerEndpoint: remoteLVMEndpoint, Log: remoteLog}
	tailLog(t, &req)
}

func TestRemoteDownloadLog(t *testing.T) {
	req := common.TailLogRequest{LogViewerEndpoint: remoteLVMEndpoint, Log: remoteLog}
	downloadLog(t, &req)
}

func TestRemoteErrors(t *testing.T) {
	req := common.ErrorsRequest{
		LogViewerEndpoint: remoteLVMEndpoint,
		StatsRequest:      &model.StatsRequest{Log: remoteLog, LogStructure: ls},
		From:              0,
		Size:              100,
	}
	errors(t, &req)
}

func TestRemoteStats(t *testing.T) {
	req := common.StatsRequest{
		StatsRequest: &model.StatsRequest{Log: remoteLog,
			LogStructure: ls},
		LogViewerEndpoint: remoteLVMEndpoint,
	}
	stats(t, &req)
}

func TestCollectRemoteStats(t *testing.T) {
	req := common.CollectStatsRequest{
		StatsRequest: &common.StatsRequest{
			LogViewerEndpoint: remoteLVMEndpoint,
			StatsRequest: &model.StatsRequest{
				Log:          remoteLog,
				LogStructure: ls},
		},
		Date: "2021-04-26",
	}
	collectStats(t, &req)
}

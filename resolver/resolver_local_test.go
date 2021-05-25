package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/common"
)

var (
	remoteLVMEndpoint = fmt.Sprintf("%v/iq-logviewer/lvm", "https://10.106.11.95:8090")
	localLVMEndpoint  = getLocalEndpoint()
	remoteLog         = "test-logs/java-app.log"
	localLog          = "../test-logs/java-app.log"
	ls                = &model.LogStructure{Date: 0, Level: 2, Message: 6, Reqid: 5, User: 4, DateFormat: "2006-01-02"}
)

func getLocalEndpoint() string {
	host := getHostname()
	return fmt.Sprintf("https://%v:8090/iq-logviewer/lvm", host)
}

func TestIsLocal(t *testing.T) {
	host := getHostname()
	url := fmt.Sprintf("https://%v:8090/iq-logviewer", host)
	fmt.Println(url)
	res := isLocal(context.Background(), url)
	if !res {
		t.Fatal("Should be local")
	}
	url = fmt.Sprintf("https://%v.nam.com:8090/iq-logviewer", host)
	fmt.Println(url)
	res = isLocal(context.Background(), url)
	if !res {
		t.Fatal("Should be local")
	}
	url = "https://some.remote.host:8090/iq-logviewer"
	fmt.Println(url)
	res = isLocal(context.Background(), url)
	if res {
		t.Fatal("Should be remote")
	}
}

func TestLocalSearch(t *testing.T) {
	hosts := make([]common.HostDetails, 0, 1)
	hosts = append(hosts, common.HostDetails{LogViewerEndpoint: localLVMEndpoint, Logs: []string{localLog}})
	sr := common.SearchRequest{Hosts: hosts, Value: "bc23456"}
	search(t, &sr)
}

func TestLocalListLogs(t *testing.T) {
	host := common.HostDetails{LogViewerEndpoint: localLVMEndpoint, Logs: []string{localLog}}
	listLogs(t, &host)
}

func TestLocalTailLog(t *testing.T) {
	req := common.TailLogRequest{LogViewerEndpoint: localLVMEndpoint, Log: localLog}
	tailLog(t, &req)
}

func TestLocalDownloadLog(t *testing.T) {
	req := common.TailLogRequest{LogViewerEndpoint: localLVMEndpoint, Log: localLog}
	downloadLog(t, &req)
}

func downloadLog(t *testing.T, r *common.TailLogRequest) {
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/download-log", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := DownloadLog(rr, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.([]byte)) == 0 {
		t.Fatal("empty")
	}
}

func TestLocalStats(t *testing.T) {
	req := common.StatsRequest{
		StatsRequest: &model.StatsRequest{Log: localLog,
			LogStructure: ls},
		LogViewerEndpoint: localLVMEndpoint,
	}
	stats(t, &req)
}

func TestLocalCollectStats(t *testing.T) {
	req := common.CollectStatsRequest{
		StatsRequest: &common.StatsRequest{
			LogViewerEndpoint: localLVMEndpoint,
			StatsRequest: &model.StatsRequest{
				Log:          localLog,
				LogStructure: ls},
		},
		Date: "2021-04-26",
	}
	collectStats(t, &req)
}

func collectStats(t *testing.T, sr *common.CollectStatsRequest) {
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/collect-stats", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := CollectStatsHandler(rr, req)
	if err != nil {
		t.Fatal(err)
	}

	var csr model.CollectStatsRsults
	switch res.(type) {
	case []byte:
		if err = json.Unmarshal(res.([]byte), &csr); err != nil {
			t.Fatalf("Could not serialize bytes, %v", err)
		}
	default:
		csr = *(res.(*model.CollectStatsRsults))
	}
	if csr.TotalRequests != 5 {
		t.Fatal("Should be 5 requests")
	}
	if len(csr.Users) != 2 {
		t.Fatal("Should find 2 users")
	}
	levels := csr.Users["bc23456"]
	if levels["ERROR"] != 2 {
		t.Fatal("should have 2 errors")
	}
	if levels["INFO"] != 1 {
		t.Fatal("should have 1 info")
	}
}

func TestLocalErrors(t *testing.T) {
	req := common.ErrorsRequest{
		LogViewerEndpoint: localLVMEndpoint,
		StatsRequest:      &model.StatsRequest{Log: localLog, LogStructure: ls},
		From:              0,
		Size:              100,
	}
	errors(t, &req)
}

func errors(t *testing.T, er *common.ErrorsRequest) {
	b, err := json.Marshal(er)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/errors", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := Errors(rr, req)
	if err != nil {
		t.Fatal(err)
	}

	var details model.ErrorDetailsPagination
	switch res.(type) {
	case []byte:
		if err = json.Unmarshal(res.([]byte), &details); err != nil {
			t.Fatalf("Could not serialize bytes, %v", err)
		}
	default:
		details = *(res.(*model.ErrorDetailsPagination))
	}
	if len(details.ErrorDetails) == 0 {
		t.Fatal("should not be empty")
	}
}

func stats(t *testing.T, sr *common.StatsRequest) {
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/stats", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := Stats(rr, req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]*model.Stat
	switch res.(type) {
	case []byte:
		json.Unmarshal(res.([]byte), &m)
	default:
		m = res.(map[string]*model.Stat)

	}
	if len(m) == 0 {
		t.Fatal("should not be empty")
	}
}

func tailLog(t *testing.T, r *common.TailLogRequest) {
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/tail-log", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := TailLog(rr, req)
	if err != nil {
		t.Fatal(err)
	}
	tlr := res.(*model.TailLogResponse)
	if len(tlr.Lines) == 0 {
		t.Fatal("empty")
	}
}

func listLogs(t *testing.T, hd *common.HostDetails) {
	b, err := json.Marshal(hd)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/list-logs", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := ListLogs(rr, req)
	if err != nil {
		t.Fatal(err)
	}
	responses := res.([]model.LogDetails)
	if len(responses) == 0 {
		t.Fatal("no matches")
	}
}

func search(t *testing.T, sr *common.SearchRequest) {
	b, err := json.Marshal(sr)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/search", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	res, err := Search(rr, req)
	if err != nil {
		t.Fatal(err)
	}
	responses := res.([]model.GrepResponse)
	if len(responses) == 0 || len(responses[0].Lines) == 0 {
		t.Fatal("no matches")
	}

}

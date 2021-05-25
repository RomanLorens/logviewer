package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/RomanLorens/logviewer-module/api"
	h "github.com/RomanLorens/logviewer-module/handler"
	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/common"
	"github.com/RomanLorens/logviewer/httpclient"
	l "github.com/RomanLorens/logviewer/logger"
)

var (
	logger   = l.L
	lvh      = h.NewHandler(logger)
	lapi     = api.NewLocalAPI(logger)
	hostname = getHostname()
)

func getHostname() string {
	cmd := exec.Command("hostname")
	out, err := cmd.CombinedOutput()
	var h string
	if err != nil {
		logger.Error(context.Background(), "hostname command failed, %v", err)
	} else {
		h = strings.TrimSpace(string(out))
	}
	if len(h) == 0 {
		//todo
	}
	logger.Info(context.Background(), "Resolved hostname as '%v'", h)
	return h
}

//Search search
func Search(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.SearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	out := make([]model.GrepResponse, 0)
	for _, h := range req.Hosts {
		host := parseHostName(r.Context(), h.LogViewerEndpoint)
		if isLocal(r.Context(), h.LogViewerEndpoint) {
			res := lapi.Grep(r.Context(), &model.GrepRequest{Value: req.Value, Logs: h.Logs})
			for i := range res {
				res[i].Host = host
			}
			out = append(out, res...)
		} else {
			url := httpclient.BuildURL(h.LogViewerEndpoint, model.SearchEndpoint)
			bytes, err := httpclient.Request(r.Context(), url, &model.GrepRequest{Value: req.Value, Logs: h.Logs}, r.Header)
			if err != nil {
				continue
			}
			var gr []model.GrepResponse
			if err := json.Unmarshal(bytes, &gr); err != nil {
				logger.Error(r.Context(), "Could not unmarshal remote response, %v", err)
				continue
			}
			for i := range gr {
				gr[i].Host = host
			}
			out = append(out, gr...)
		}
	}
	return out, nil
}

//ListLogs list ologs
func ListLogs(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.HostDetails
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	host := parseHostName(r.Context(), req.LogViewerEndpoint)
	if isLocal(r.Context(), req.LogViewerEndpoint) {
		lds := lapi.ListLogs(r.Context(), &model.ListLogsRequest{Logs: req.Logs})
		for i := range lds {
			lds[i].Host = host
		}
		return lds, nil
	}
	url := httpclient.BuildURL(req.LogViewerEndpoint, model.ListLogsEndpoint)
	res, err := httpclient.Request(r.Context(), url, &model.ListLogsRequest{Logs: req.Logs}, r.Header)
	if err != nil {
		return nil, err
	}
	var lds []model.LogDetails
	if err := json.Unmarshal(res, &lds); err != nil {
		return nil, err
	}
	for i := range lds {
		lds[i].Host = host
	}
	return lds, err
}

//TailLog tail log
func TailLog(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.TailLogRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	host := parseHostName(r.Context(), req.LogViewerEndpoint)
	if isLocal(r.Context(), req.LogViewerEndpoint) {
		res, err := lapi.TailLog(r.Context(), req.Log)
		if err != nil {
			return nil, err
		}
		res.Host = host
		return res, nil
	}

	url := httpclient.BuildURL(req.LogViewerEndpoint, model.TailLogEndpoint)
	res, err := httpclient.Request(r.Context(), url, &model.LogRequest{Log: req.Log}, r.Header)
	if err != nil {
		return nil, err
	}
	var tlr model.TailLogResponse
	if err := json.Unmarshal(res, &tlr); err != nil {
		return nil, err
	}
	tlr.Host = host
	return &tlr, nil
}

//DownloadLog download log
func DownloadLog(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.TailLogRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	if isLocal(r.Context(), req.LogViewerEndpoint) {
		return lapi.DownloadLog(req.Log)
	}
	url := httpclient.BuildURL(req.LogViewerEndpoint, model.DownloadLogEndpoint)
	return httpclient.Request(r.Context(), url, &model.LogRequest{Log: req.Log}, r.Header)
}

//CollectStatsHandler collect stats
func CollectStatsHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var sr common.CollectStatsRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	return CollectStats(r.Context(), &sr, r.Header)
}

//CollectStats collect stats
func CollectStats(ctx context.Context, req *common.CollectStatsRequest, headers http.Header) (*model.CollectStatsRsults, error) {
	csr := model.CollectStatsRequest{StatsRequest: &model.StatsRequest{Log: req.Log, LogStructure: req.LogStructure}, Date: req.Date}
	if isLocal(ctx, req.LogViewerEndpoint) {
		return lapi.CollectStats(ctx, &csr)
	}
	url := httpclient.BuildURL(req.LogViewerEndpoint, model.CollectStatsEndpoint)
	bytes, err := httpclient.Request(ctx, url, &csr, headers)
	if err != nil {
		return nil, err
	}
	var res model.CollectStatsRsults
	if err = json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

//Errors errors
func Errors(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.ErrorsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	if isLocal(r.Context(), req.LogViewerEndpoint) {
		return lapi.Errors(r.Context(), &model.ErrorsRequest{
			From:         req.From,
			Size:         req.Size,
			StatsRequest: req.StatsRequest,
		})
	}

	url := httpclient.BuildURL(req.LogViewerEndpoint, model.ErrorsEndpoint)
	bytes, err := httpclient.Request(r.Context(), url, &model.ErrorsRequest{
		From:         req.From,
		Size:         req.Size,
		StatsRequest: req.StatsRequest,
	}, r.Header)
	if err != nil {
		return nil, err
	}
	var res model.ErrorDetailsPagination
	if err = json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

//Stats stats
func Stats(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var sr common.StatsRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	if isLocal(r.Context(), sr.LogViewerEndpoint) {
		return lapi.Stats(r.Context(), &model.StatsRequest{Log: sr.Log, LogStructure: sr.LogStructure})
	}

	url := httpclient.BuildURL(sr.LogViewerEndpoint, model.StatsEndpoint)
	bytes, err := httpclient.Request(r.Context(), url, &model.StatsRequest{Log: sr.Log, LogStructure: sr.LogStructure}, r.Header)
	if err != nil {
		return nil, err
	}
	var res map[string]*model.Stat
	if err = json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func parseHostName(ctx context.Context, _url string) string {
	u, err := url.Parse(_url)
	if err != nil {
		logger.Error(ctx, "Could not parse %v, %v", _url, err)
		return ""
	}
	return u.Hostname()
}

func isLocal(ctx context.Context, logviewerURL string) bool {
	u, err := url.Parse(logviewerURL)
	if err != nil {
		logger.Error(ctx, "Could not parse %v, %v", logviewerURL, err)
		return true
	}
	res := strings.Contains(u.Hostname(), hostname)
	logger.Info(ctx, "'%v' resolved as local '%v'", logviewerURL, res)
	return res
}

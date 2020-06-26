package search

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	e "github.com/RomanLorens/logviewer/error"
	log "github.com/RomanLorens/logviewer/logger"
)

//RemoteSearch remote search
type RemoteSearch struct{}

//trust all - mostly for requests from localhost
var client = &http.Client{Transport: &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}}

//Tail tail log
func (RemoteSearch) Tail(ctx context.Context, app *Application) (*Result, *e.Error) {
	log.Info(ctx, "Tail log remotely")
	var res *Result
	url := apiURL(app.Host, "tail-log")
	body, err := callAPI(ctx, url, app)
	if err != nil {
		return nil, err
	}
	if er := json.Unmarshal(body, &res); er != nil {
		return nil, e.Errorf(500, "Could not read unmarshal, %v", er)
	}
	return res, nil
}

//Grep grep logs
func (RemoteSearch) Grep(ctx context.Context, url string, s *Search) ([]*Result, *e.Error) {
	log.Info(ctx, "Grep log remotely")
	url = apiURL(url, "search")
	body, err := callAPI(ctx, url, s)
	if err != nil {
		return nil, err
	}
	r := make([]*Result, len(s.Logs))
	if er := json.Unmarshal(body, &r); er != nil {
		return r, e.Errorf(500, "Could not read unmarshal, %v", er)
	}
	return r, nil
}

//List list logs
func (RemoteSearch) List(ctx context.Context, url string, s *Search) ([]*LogDetails, *e.Error) {
	var logs []*LogDetails
	url = apiURL(url, "list-logs")
	body, err := callAPI(ctx, url, s)
	if err != nil {
		return nil, err
	}
	if er := json.Unmarshal(body, &logs); er != nil {
		return nil, e.Errorf(500, "Could not read unmarshal, %v", er)
	}
	return logs, nil
}

func callAPI(ctx context.Context, url string, post interface{}) ([]byte, *e.Error) {
	log.Info(ctx, "Remote api for %v", url)
	b, err := json.Marshal(post)
	if err != nil {
		return nil, e.Errorf(500, "Could not marshal post %v", err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, e.Errorf(500, "Request to %v failed, %v", url, err)
	}
	log.Info(ctx, "Api response %v", resp)
	if resp.StatusCode != 200 {
		return nil, e.Errorf(resp.StatusCode, "Request to %v failed, %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, e.Errorf(500, "Could not read response, %v", err)
	}
	return body, nil
}

func apiURL(url string, api string) string {
	if strings.HasSuffix(url, api) {
		return url
	}
	if url[len(url)-1:] != "/" {
		url = url + "/"
	}
	return url + api
}

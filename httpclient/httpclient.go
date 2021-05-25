package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	l "github.com/RomanLorens/logviewer/logger"
)

var (
	//trust all
	client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	logger = l.L
)

//Request make req
func Request(ctx context.Context, url string, post interface{}, headers http.Header) ([]byte, error) {
	logger.Info(ctx, "Remote api for %v", url)
	b, err := json.Marshal(post)
	if err != nil {
		logger.Error(ctx, "Could not marshal post %v", err)
		return nil, fmt.Errorf("Could not marshal post %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		logger.Error(ctx, "Could not create req for %v, %v", url, err)
		return nil, fmt.Errorf("Could not create req for %v, %v", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Request to %v failed, %v", url, err)
		return nil, fmt.Errorf("Request to %v failed, %v", url, err)
	}
	defer resp.Body.Close()
	logger.Info(ctx, "Api response %v", resp)
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		logger.Error(ctx, "Request to %v failed, %v", url, string(b))
		bb, err := json.Marshal(post)
		if err == nil {
			logger.Error(ctx, "Request body '%v'", string(bb))
		}
		return nil, fmt.Errorf("Request to %v failed, %v", url, string(b))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, "Could not read response, %v", err)
		return nil, fmt.Errorf("Could not read response, %v", err)
	}
	return body, nil
}

//BuildURL build url
func BuildURL(url string, api string) string {
	if strings.HasSuffix(url, api) {
		return url
	}
	if url[len(url)-1:] != "/" {
		url = url + "/"
	}
	return url + api
}

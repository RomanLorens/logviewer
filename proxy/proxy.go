package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/RomanLorens/logviewer/logger"
)

//Forward forwards request to url
func Forward(url string, w *http.ResponseWriter, r *http.Request) error {
	logger.Info(r.Context(), fmt.Sprintf("proxy for %v", url))
	var body []byte
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		logger.Error(r.Context(), fmt.Sprintf("Could not read bytes %v", err.Error()))
		return fmt.Errorf("Could not read bytes %v", err.Error())
	}
	req, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	if err != nil {
		logger.Error(r.Context(), fmt.Sprintf("Could not create request for %v, %v", url, err.Error()))
		return fmt.Errorf("Could not create request for %v, %v", url, err.Error())
	}
	client := &http.Client{}
	for k, v := range r.Header {
		for _, header := range v {
			req.Header.Add(k, header)
		}
	}
	res, err := client.Do(req)
	if err != nil {
		e := fmt.Errorf("Error from client %v, %v", url, err.Error())
		logger.Error(r.Context(), e.Error())
		return e
	}
	logger.Info(r.Context(), fmt.Sprintf("status from %v - %v", url, res.StatusCode))

	for k, v := range res.Header {
		for _, header := range v {
			(*w).Header().Add(k, header)
		}
	}
	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if !(res.StatusCode >= 200 && res.StatusCode < 299) {
		logger.Error(r.Context(), fmt.Sprintf("headers: %v", res.Header))
		s, err := gunzipWrite(b)
		if err != nil {
			s = string(b)
		}
		logger.Error(r.Context(), fmt.Sprintf("Error from cp: %v", s))
	}
	if err != nil {
		logger.Error(r.Context(), fmt.Sprintf("Error on reading cp response bytes %v", err.Error()))
		return fmt.Errorf("Error on reading cp response bytes %v", err.Error())
	}
	(*w).WriteHeader(res.StatusCode)
	(*w).Write(b)
	return nil
}

func gunzipWrite(data []byte) (string, error) {
	var w bytes.Buffer
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if gr != nil {
		defer gr.Close()
		data, err = ioutil.ReadAll(gr)
		if err != nil {
			return "", err
		}
	}
	w.Write(data)
	return w.String(), nil
}

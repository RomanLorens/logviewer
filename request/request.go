package request

import (
	"io/ioutil"
	"net/http"
)

//Details details
type Details struct {
	RequestHeaders map[string][]string `json:"requestHeaders"`
	Method         string              `json:"method"`
	URL            string              `json:"url"`
	Body           string              `json:"body"`
	RemoteAddress  string              `json:"remoteAddress"`
	RequestCookies map[string]string   `json:"requestCookies"`
	Host           string              `json:"host"`
}

//Parse parses request
func Parse(r *http.Request) *Details {
	m := make(map[string][]string, 0)
	c := make(map[string]string, 0)
	for name, values := range r.Header {
		for _, value := range values {
			m[name] = append(m[name], value)
		}
	}
	b, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	for _, cookie := range r.Cookies() {
		c[cookie.Name] = cookie.Value
	}

	return &Details{
		RequestHeaders: m,
		Method:         r.Method,
		URL:            "https://" + r.Host + r.RequestURI,
		Body:           string(b),
		RemoteAddress:  r.RemoteAddr,
		RequestCookies: c,
		Host:           r.Host,
	}
}

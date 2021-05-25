package handler

import (
	"net/http"

	log "github.com/RomanLorens/logger/log"
	"github.com/RomanLorens/logviewer/config"
	l "github.com/RomanLorens/logviewer/logger"
	f "github.com/RomanLorens/rl-common/filter"
)

//UserFilter auth by user or ip filter
type UserFilter struct{}

var (
	//IPFilterInstance ip filter
	IPFilterInstance = f.NewBearerTokenIPFilter(l.L, config.Config.WhiteListIPs)
	//UserFilterInstance user filter filter
	UserFilterInstance = &UserFilter{}
)

//UserFilter authorize by user or whitelisted ips
func (UserFilter) DoFilter(r *http.Request) (bool, *http.Request) {
	user := r.Context().Value(log.UserKey)
	if user == "anonymous" {
		return IPFilterInstance.DoFilter(r)
	}
	return true, r
}

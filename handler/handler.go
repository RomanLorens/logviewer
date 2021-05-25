package handler

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/RomanLorens/logger/log"
	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/auth"
	"github.com/RomanLorens/logviewer/common"
	"github.com/RomanLorens/logviewer/config"
	l "github.com/RomanLorens/logviewer/logger"
	"github.com/RomanLorens/logviewer/resolver"
	"github.com/RomanLorens/logviewer/scheduler"
	"github.com/RomanLorens/logviewer/user"
	f "github.com/RomanLorens/rl-common/filter"

	h "github.com/RomanLorens/logviewer-module/handler"

	"github.com/gorilla/mux"
	uuid "github.com/nu7hatch/gouuid"
)

var (
	logger = l.L
	lvm    = h.NewHandler(logger)
)

type errorJSON struct {
	Msg   string `json:"msg"`
	ReqID string `json:"reqid"`
}

//StartServer inits and starts server
func StartServer() {
	r := mux.NewRouter()
	r.Use(loggingFilter)
	r.NotFoundHandler = http.HandlerFunc(notFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(notFound)

	r.PathPrefix("/iq-logviewer-ui/").
		Handler(http.StripPrefix("/iq-logviewer-ui/", http.FileServer(http.Dir(config.Config.ServerConfiguration.StaticFolder))))
	logger.Info(context.Background(), "Registered /logviewer-ui/ with static folder %v ", config.Config.ServerConfiguration.StaticFolder)

	register("/", root, r, http.MethodGet)
	register("/"+model.SearchEndpoint, resolver.Search, r, http.MethodPost)
	register("/"+model.ListLogsEndpoint, resolver.ListLogs, r, http.MethodPost)
	register("/"+model.TailLogEndpoint, resolver.TailLog, r, http.MethodPost)
	register("/"+model.StatsEndpoint, resolver.Stats, r, http.MethodPost)
	register("/"+model.ErrorsEndpoint, resolver.Errors, r, http.MethodPost)
	register("/"+model.DownloadLogEndpoint, resolver.DownloadLog, r, http.MethodPost)
	register("/"+model.CollectStatsEndpoint, resolver.CollectStatsHandler, r, http.MethodPost)

	register("/lvm/"+model.SearchEndpoint, lvm.Search, r, http.MethodPost)
	register("/lvm/"+model.ListLogsEndpoint, lvm.ListLogs, r, http.MethodPost)
	register("/lvm/"+model.TailLogEndpoint, lvm.TailLog, r, http.MethodPost)
	register("/lvm/"+model.StatsEndpoint, lvm.Stats, r, http.MethodPost)
	register("/lvm/"+model.ErrorsEndpoint, lvm.Errors, r, http.MethodPost)
	register("/lvm/"+model.DownloadLogEndpoint, lvm.DownloadLog, r, http.MethodPost)
	register("/lvm/"+model.CollectStatsEndpoint, lvm.CollectStats, r, http.MethodPost)

	register("/auth/current-user", currentUser, r, http.MethodGet)
	register("/user-details", userDetailsHandler, r, http.MethodGet)
	registerWithFilters("/populate-stats", []f.Filter{IPFilterInstance}, populateAppStats, r, http.MethodPost)
	registerWithFilters("/populate-stats-batch", []f.Filter{IPFilterInstance}, populateStatsBatch, r, http.MethodGet)
	register("/app-stats", appStatsHandler, r, http.MethodPost)
	register("/config", appsConfigHandler, r, http.MethodGet)

	supportHandlers(r)

	registerWS("/ws/apps-health", lvm.AppsHealth, r)
	registerWS("/ws/tail-log", lvm.TailLogWS, r)

	if config.Config.EnableScheduler {
		scheduler.InitScheduler()
	}

	cert := config.Config.ServerConfiguration.Cert
	if cert != "" {
		logger.Info(context.Background(), "Starting https server on %v port, context %v", config.Config.ServerConfiguration.Port, config.Config.ServerConfiguration.Context)
		cfg := &tls.Config{
			MinVersion:       tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			/*PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			*/
		}
		srv := &http.Server{
			Addr:      fmt.Sprintf(":%v", config.Config.ServerConfiguration.Port),
			TLSConfig: cfg,
			Handler:   r,
		}
		if err := srv.ListenAndServeTLS(cert, config.Config.ServerConfiguration.CertKey); err != nil {
			logger.Error(context.Background(), "Error on server main thread, %v", err)
		}
	} else {
		logger.Info(context.Background(), "Starting server on %v port, context %v", config.Config.ServerConfiguration.Port, config.Config.ServerConfiguration.Context)
		if err := http.ListenAndServe(fmt.Sprintf(":%v", config.Config.ServerConfiguration.Port), r); err != nil {
			logger.Error(context.Background(), "Error on server main thread, %v", err)
		}
	}
}

func appStatsHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var req common.StatReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	return config.GetAppStats(r.Context(), &req)
}

func populateStatsBatch(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	date := r.FormValue("date")
	if date == "" {
		return nil, fmt.Errorf("Must pass date")
	}
	return scheduler.PopulateStatsBatch(r.Context(), date)
}

func populateAppStats(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var s common.AppCollectStatsRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		return nil, fmt.Errorf("Could not parse req body, %v", err)
	}
	stats, er := config.GetStatsKeys(r.Context(), s.Date)
	if er != nil {
		return nil, er
	}
	key := common.StatsKey(&common.Stats{App: s.App, Env: s.Env, LogPath: s.Log, Date: s.Date})
	if _, exists := stats[key]; exists {
		return fmt.Sprintf("Stats for %v already exists in mongo", s), nil
	}
	csr := common.CollectStatsRequest{StatsRequest: &common.StatsRequest{LogViewerEndpoint: s.LogViewerEndpoint,
		StatsRequest: &model.StatsRequest{Log: s.Log, LogStructure: s.LogStructure}}, Date: s.Date}
	res, err := resolver.CollectStats(r.Context(), &csr, r.Header)
	if er != nil {
		return nil, er
	}
	er = config.SaveStats(r.Context(), &common.Stats{Stats: res, LogPath: s.Log, App: s.App, Env: s.Env, Date: s.Date})
	if er != nil {
		return nil, er
	}
	return fmt.Sprintf("Saved stat %v", res), nil
}

func userDetailsHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return user.Details(r.Context(), r.FormValue("user"))
}

func currentUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return auth.UserWithRoles(r), nil
}

func appsConfigHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return config.Config.ApplicationsConfig, nil
}

func notFound(w http.ResponseWriter, r *http.Request) {
	r = setContext(w, r)
	logger.Error(r.Context(), "Not found [%v] %v, ip: %v", r.Method, r.URL.RequestURI(), r.RemoteAddr)
	w.WriteHeader(http.StatusNotFound)
}

func root(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return "OK", nil
}

func registerWithFilters(path string, filters []f.Filter, fn func(w http.ResponseWriter, r *http.Request) (interface{}, error),
	r *mux.Router, methods ...string) {
	_register(path, filters, fn, r, methods...)
}

func register(path string, fn func(w http.ResponseWriter, r *http.Request) (interface{}, error),
	r *mux.Router, methods ...string) {
	_register(path, []f.Filter{UserFilterInstance}, fn, r, methods...)
}

func _register(path string, filters []f.Filter, fn func(w http.ResponseWriter, r *http.Request) (interface{}, error),
	r *mux.Router, methods ...string) {
	endpoint := fmt.Sprintf("%s%s", config.Config.ServerConfiguration.Context, path)
	if endpoint[0] != '/' {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		if filters != nil {
			for _, f := range filters {
				ok, r := f.DoFilter(r)
				if !ok {
					errorResponse(fmt.Errorf("Unauthorized by filter"), w, r)
					return
				}
			}
		}
		res, err := fn(w, r)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			errorResponse(err, w, r)
			return
		}
		logger.Info(r.Context(), "[%v] %v completed", r.Method, r.URL.RequestURI())
		if res == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(res); nil != err {
			errorResponse(fmt.Errorf("Could not encode response"), w, r)
		}
	}
	r.HandleFunc(endpoint, h).Methods(methods...)
}

func registerWS(path string, fn func(w http.ResponseWriter, r *http.Request) error,
	r *mux.Router) {
	endpoint := fmt.Sprintf("%s%s", config.Config.ServerConfiguration.Context, path)
	if endpoint[0] != '/' {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}
	h := func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			logger.Error(r.Context(), err.Error())
		}

	}
	r.HandleFunc(endpoint, h)
}

func errorResponse(err error, w http.ResponseWriter, r *http.Request) {
	logger.Error(r.Context(), err.Error())
	logger.Error(r.Context(), "[%v] %v failed", r.Method, r.URL.RequestURI())
	e := errorJSON{Msg: err.Error()}
	id := r.Context().Value(log.ReqID)
	if id != nil {
		e.ReqID = id.(string)
	}
	w.WriteHeader(500)
	if er := json.NewEncoder(w).Encode(e); er != nil {
		logger.Error(r.Context(), "error was not serialized, %v", err)
	}
}

func loggingFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = setContext(w, r)
		next.ServeHTTP(w, r)
	})
}

func setContext(w http.ResponseWriter, r *http.Request) *http.Request {
	id := r.Header.Get("x-citiportal-requestid")
	if len(id) == 0 {
		if v, err := uuid.NewV4(); err == nil {
			id = v.String()
		}
	}
	u := auth.UserFromRequest(r)
	ctx := context.WithValue(r.Context(), log.UserKey, u)
	ctx = context.WithValue(ctx, log.ReqID, id)
	r = r.WithContext(ctx)
	w.Header().Add("__req_id__", id)
	logger.Info(r.Context(), "[%v] %v", r.Method, r.URL.RequestURI())
	return r
}

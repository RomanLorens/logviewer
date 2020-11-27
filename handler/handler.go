package handler

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/RomanLorens/logviewer/auth"
	"github.com/RomanLorens/logviewer/config"
	e "github.com/RomanLorens/logviewer/error"
	log "github.com/RomanLorens/logviewer/logger"
	"github.com/RomanLorens/logviewer/proxy"
	"github.com/RomanLorens/logviewer/request"
	"github.com/RomanLorens/logviewer/search"
	"github.com/RomanLorens/logviewer/stat"
	"github.com/RomanLorens/logviewer/user"

	"github.com/gorilla/mux"
	uuid "github.com/nu7hatch/gouuid"
)

//StartServer inits and starts server
func StartServer() {
	r := mux.NewRouter()
	r.Use(loggingFilter)
	r.NotFoundHandler = http.HandlerFunc(notFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(notFound)

	r.PathPrefix("/iq-logviewer-ui/").
		Handler(http.StripPrefix("/iq-logviewer-ui/", http.FileServer(http.Dir(config.ServerConfiguration.StaticFolder))))
	log.Info(context.Background(), "Registered /logviewer-ui/ with static folder %v ", config.ServerConfiguration.StaticFolder)

	register("/", root, r, http.MethodGet)
	register("/search", searchHandler, r, http.MethodPost)
	register("/config", configHandler, r, http.MethodGet)
	register("/list-logs", listLogs, r, http.MethodPost)
	register("/tail-log", tailLog, r, http.MethodPost)
	register("/stats", stats, r, http.MethodPost)
	register("/errors", errors, r, http.MethodPost)
	register("/support/health", health, r, http.MethodGet)
	register("/auth/current-user", currentUser, r, http.MethodGet)
	register("/support/request-details", printRequest, r, http.MethodGet, http.MethodPost)
	register("/user-details", userDetailsHandler, r, http.MethodGet)

	registerWithoutResponse("/download-log", downloadLog, r, http.MethodPost)
	registerWithoutResponse("/support/proxy", proxyHandler, r, http.MethodGet, http.MethodPost)
	registerWS("/ws/apps-health", appsHealth, r)
	registerWS("/ws/tail-log", tailLogWS, r)

	cert := config.ServerConfiguration.Cert
	if cert != "" {
		log.Info(context.Background(), "Starting https server on %v port, context %v", config.ServerConfiguration.Port, config.ServerConfiguration.Context)
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
			Addr:      fmt.Sprintf(":%v", config.ServerConfiguration.Port),
			TLSConfig: cfg,
			Handler:   r,
		}
		if err := srv.ListenAndServeTLS(cert, config.ServerConfiguration.CertKey); err != nil {
			log.Error(context.Background(), "Error on server main thread, %v", err)
		}
	} else {
		log.Info(context.Background(), "Starting server on %v port, context %v", config.ServerConfiguration.Port, config.ServerConfiguration.Context)
		if err := http.ListenAndServe(fmt.Sprintf(":%v", config.ServerConfiguration.Port), r); err != nil {
			log.Error(context.Background(), "Error on server main thread, %v", err)
		}
	}

}

func userDetailsHandler(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return user.Details(r.Context(), r.FormValue("user"))
}

func currentUser(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return auth.UserWithRoles(r), nil
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	proxy.Forward(r.FormValue("url"), &w, r)
}

func downloadLog(w http.ResponseWriter, r *http.Request) {
	var ld search.LogDownload
	err := json.NewDecoder(r.Body).Decode(&ld)
	if err != nil {
		errorResponse(e.Errorf(500, err.Error()), w, r)
		return
	}
	defer r.Body.Close()
	b, er := search.DownloadLog(r.Context(), &ld)
	if er != nil {
		errorResponse(er, w, r)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", path.Base(ld.Log)))
	w.Write(b)
}

func health(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return "OK", nil
}

func printRequest(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return request.Parse(r), nil
}

func stats(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	app, err := toApp(r)
	if err != nil {
		return nil, err
	}
	return stat.Get(r.Context(), app)
}

func errors(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	app, err := toApp(r)
	if err != nil {
		return nil, err
	}
	return stat.GetErrors(r.Context(), app)
}

func tailLog(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	app, err := toApp(r)
	if err != nil {
		return nil, err
	}
	return search.TailLog(r.Context(), app)
}

func listLogs(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	var s, err = toSearch(r)
	if err != nil {
		return nil, err
	}
	return search.ListLogs(r.Context(), s)
}

func toApp(r *http.Request) (*search.Application, *e.Error) {
	var app search.Application
	bytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not reead req body, %v", err)
	}
	err = json.Unmarshal(bytes, &app)
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not unmarshal data, %v", err)
	}
	return &app, nil
}

func configHandler(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return config.ApplicationsConfig, nil
}

func searchHandler(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	var s, err = toSearch(r)
	if err != nil {
		return nil, err
	}
	res, er := search.Find(r.Context(), s)
	return res, er
}

func toSearch(r *http.Request) (*search.Search, *e.Error) {
	var s search.Search
	bytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not reead req body, %v", err)
	}
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not unmarshal data, %v", err)
	}
	return &s, nil
}

func notFound(w http.ResponseWriter, r *http.Request) {
	r = setContext(w, r)
	log.Error(r.Context(), "Not found [%v] %v", r.Method, r.URL.RequestURI())
	w.WriteHeader(http.StatusNotFound)
}

func root(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error) {
	return "OK", nil
}

func register(path string, fn func(w http.ResponseWriter, r *http.Request) (interface{}, *e.Error),
	r *mux.Router, methods ...string) {
	endpoint := fmt.Sprintf("%s%s", config.ServerConfiguration.Context, path)
	if endpoint[0] != '/' {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}
	log.Info(context.Background(), "Registered endpoint %s %v", endpoint, methods)

	h := func(w http.ResponseWriter, r *http.Request) {
		res, err := fn(w, r)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			errorResponse(err, w, r)
		}
		if err := json.NewEncoder(w).Encode(res); nil != err {
			errorResponse(e.Errorf(http.StatusInternalServerError, "Could not encode response"), w, r)
		}
	}
	r.HandleFunc(endpoint, h).Methods(methods...)
}

func registerWithoutResponse(path string, fn func(w http.ResponseWriter, r *http.Request),
	r *mux.Router, methods ...string) {
	endpoint := fmt.Sprintf("%s%s", config.ServerConfiguration.Context, path)
	if endpoint[0] != '/' {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}
	log.Info(context.Background(), "Registered endpoint %s %v", endpoint, methods)
	r.HandleFunc(endpoint, fn).Methods(methods...)
}

func registerWS(path string, fn func(w http.ResponseWriter, r *http.Request) *e.Error,
	r *mux.Router) {
	endpoint := fmt.Sprintf("%s%s", config.ServerConfiguration.Context, path)
	if endpoint[0] != '/' {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}
	log.Info(context.Background(), "Registered WS endpoint %s", endpoint)
	h := func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			log.Error(r.Context(), err.Message)
		}

	}
	r.HandleFunc(endpoint, h)
}

func errorResponse(err *e.Error, w http.ResponseWriter, r *http.Request) {
	log.Error(r.Context(), err.Message)
	w.WriteHeader(err.StatusCode)
	if er := json.NewEncoder(w).Encode(err); er != nil {
		log.Error(r.Context(), "error was not serialized, %v", err)
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
	log.Info(r.Context(), "request [%v] %v", r.Method, r.URL.RequestURI())
	return r
}

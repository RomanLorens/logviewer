package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/RomanLorens/logviewer-module/utils"
	"github.com/RomanLorens/logviewer/config"
	"github.com/RomanLorens/logviewer/request"
	f "github.com/RomanLorens/rl-common/filter"
	"github.com/gorilla/mux"
)

func supportHandlers(r *mux.Router) {
	register("/support/health", lvm.HealthHandler, r, http.MethodGet)
	register("/support/request-details", printRequest, r, http.MethodGet, http.MethodPost)
	registerWithFilters("/support/reload-config", []f.Filter{IPFilterInstance}, reloadConfig, r, http.MethodGet)
	registerWithFilters("/support/config", []f.Filter{IPFilterInstance}, getConfig, r, http.MethodGet)
	registerWithFilters("/support/update-config", []f.Filter{IPFilterInstance}, updateConfig, r, http.MethodPost)
	registerWithFilters("/support/stop-server", []f.Filter{IPFilterInstance}, stopServer, r, http.MethodGet)
	registerWithFilters("/support/mem-diagnostics", []f.Filter{IPFilterInstance}, lvm.MemoryDiagnostics, r, http.MethodGet)
	register("/support/proxy", lvm.ProxyHandler, r, http.MethodGet, http.MethodPost)
	register("/support/version", version, r, http.MethodGet)
}

func version(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	m := make(map[string]interface{})
	m["hostname"], _ = utils.Hostname()
	return m, nil
}

func printRequest(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return request.Parse(r), nil
}

func reloadConfig(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return config.Reload(r.Context())
}

func getConfig(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return config.Config, nil
}

func updateConfig(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var c config.AppConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return fmt.Errorf("could not decode body, %v", err), nil
	}
	defer r.Body.Close()
	return config.UpdateAppConfig(r.Context(), &c)
}

func stopServer(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	pid := os.Getpid()
	logger.Info(r.Context(), "killing server by pid %v ...", pid)
	if pid > 1 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return nil, err
		}
		err = p.Kill()
		if err != nil {
			return nil, fmt.Errorf("Could not kill process %v, %v", pid, err)
		}
	}
	return nil, nil
}

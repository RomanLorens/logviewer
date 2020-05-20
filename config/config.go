package config

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	e "github.com/RomanLorens/logviewer/error"
	"github.com/RomanLorens/logviewer/logger"
)

//ServerConfig config
type ServerConfig struct {
	Port         int
	Context      string
	StaticFolder string
}

//Config config
type Config struct {
	Application string `json:"application"`
	Hosts       []Host `json:"hosts"`
	Env         string `json:"env"`
}

//Host host
type Host struct {
	Paths    []string `json:"paths"`
	Endpoint string   `json:"endpoint"`
}

var (
	//ServerConfiguration server config
	ServerConfiguration *ServerConfig
	//ApplicationsConfig config
	ApplicationsConfig []*Config
)

func init() {
	port := flag.Int("port", 8090, "port number")
	appContext := flag.String("context", "/iq-logviewer", "server contenxt path")
	cr := flag.String("config", "static", "config resolver")
	configFile := flag.String("configFile", "static/config.json", "config resolver")
	staticFolder := flag.String("staticFolder", "./dist", "static folder")
	flag.Parse()

	var err *e.Error

	switch *cr {
	case "static":
		logger.Info(context.Background(), "Loading static config from %v", *configFile)
		ApplicationsConfig, err = FileConfigResolver{FilePath: *configFile}.GetConfig()
	case "mongo":
		logger.Info(context.Background(), "Loading mongo config from %v", *configFile)
		ApplicationsConfig, err = MongoConfigResolver{FilePath: *configFile}.GetConfig()
	default:
		panic("unknown config option")
	}
	if err != nil {
		log.Panicf("Could not init configuration with %v, %v", *configFile, err)
	}

	ServerConfiguration = &ServerConfig{Port: *port, Context: *appContext, StaticFolder: *staticFolder}
}

//Resolver config resolver
type Resolver interface {
	GetConfig() (*Config, *e.Error)
}

//FileConfigResolver gets config from file
type FileConfigResolver struct {
	FilePath string
}

//GetConfig config
func (f FileConfigResolver) GetConfig() ([]*Config, *e.Error) {
	var c []*Config
	b, err := ioutil.ReadFile(f.FilePath)
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not read file %s, %v", f.FilePath, err)
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not unmarshal config %v", err)
	}
	return c, nil
}

package config

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/RomanLorens/logviewer-module/model"
	"github.com/RomanLorens/logviewer/common"
	l "github.com/RomanLorens/logviewer/logger"
	"github.com/RomanLorens/rl-common/filter"
)

//ServerConfig config
type ServerConfig struct {
	Port         int
	Context      string
	StaticFolder string
	Cert         string
	CertKey      string
}

//AppConfig app config
type AppConfig struct {
	ID           string              `json:"id,omitempty" bson:"id,omitempty"`
	Application  string              `json:"application"`
	CollectStats bool                `json:"collectStats"`
	Hosts        []Host              `json:"hosts"`
	Env          string              `json:"env"`
	LogStructure *model.LogStructure `json:"logStructure"`
	SupportURLs  []SupportURL        `json:"supportUrls"`
}

//Host host
type Host struct {
	Paths    []string `json:"paths"`
	Endpoint string   `json:"endpoint"`
	AppHost  string   `json:"appHost"`
}

//SupportURL support url
type SupportURL struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
}

//Configuration configuration
type Configuration struct {
	ApplicationsConfig    []AppConfig
	WhiteListIPs          []filter.WhiteList
	ServerConfiguration   *ServerConfig
	UserByLoginIDURL      string
	EnableScheduler       bool
	Bearer                string
	EmailServer           string
	StatsEmailReciepients []string
}

var (
	resolver Resolver
	//Config config
	Config *Configuration
	logger = l.L
)

func init() {
	port := flag.Int("port", 8090, "port number")
	appContext := flag.String("context", "/iq-logviewer", "server contenxt path")
	cr := flag.String("config", "static", "config resolver")
	configFile := flag.String("configFile", "static/config.json", "config resolver")
	staticFolder := flag.String("staticFolder", "./dist", "static folder")
	cert := flag.String("cert", "", "https server cert")
	certKey := flag.String("certKey", "", "https server cert key")
	enableScheduler := flag.Bool("enableScheduler", false, "run scheduler on this instance")
	flag.Parse()

	if *cert != "" && *certKey == "" {
		logger.Panicf(context.Background(), "Both cert and cert key must be set!")
	}

	switch *cr {
	case "static":
		logger.Info(context.Background(), "Loading static config from %v", *configFile)
		resolver = FileConfigResolver{FilePath: *configFile}
	case "mongo":
		logger.Info(context.Background(), "Loading mongo config from %v", *configFile)
		resolver = MongoConfigResolver{FilePath: *configFile}
	default:
		logger.Panicf(context.Background(), "unknown config option")
	}
	_config, err := resolver.GetConfig(context.Background())
	if err != nil {
		logger.Panicf(context.Background(), "Could not init configuration with %v, %v", *configFile, err)
	}

	_config.ServerConfiguration = &ServerConfig{Port: *port, Context: *appContext, StaticFolder: *staticFolder,
		Cert: *cert, CertKey: *certKey}
	logger.Info(context.Background(), "Enable scheduler = %v", *enableScheduler)
	_config.EnableScheduler = *enableScheduler

	Config = _config
}

//Resolver config resolver
type Resolver interface {
	GetConfig(ctx context.Context) (*Configuration, error)
	UpdateAppConfig(ctx context.Context, cfg *AppConfig) (interface{}, error)
	SaveStats(ctx context.Context, stats *common.Stats) error
	GetStatsKeys(ctx context.Context, date string) (map[string]int, error)
	GetAppStats(ctx context.Context, req *common.StatReq) ([]common.Stats, error)
}

//FileConfigResolver gets config from file
type FileConfigResolver struct {
	FilePath string
}

//SaveStats save stats
func (FileConfigResolver) SaveStats(ctx context.Context, stats *common.Stats) error {
	return nil
}

//UpdateAppConfig update cfg
func (f FileConfigResolver) UpdateAppConfig(ctx context.Context, cfg *AppConfig) (interface{}, error) {
	return nil, nil
}

//GetAppStats get app stats
func (FileConfigResolver) GetAppStats(ctx context.Context, req *common.StatReq) ([]common.Stats, error) {
	return nil, nil
}

//GetConfig config
func (f FileConfigResolver) GetConfig(ctx context.Context) (*Configuration, error) {
	logger.Info(ctx, "Loading config with resolver %v", f)
	var c []AppConfig
	b, err := ioutil.ReadFile(f.FilePath)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, %v", f.FilePath, err)
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("Could not unmarshal config %v", err)
	}
	return &Configuration{ApplicationsConfig: c}, nil
}

//Reload reloads config
func Reload(ctx context.Context) (*Configuration, error) {
	cfg, err := resolver.GetConfig(ctx)
	if err == nil {
		Config = cfg
	}
	return cfg, err
}

//UpdateAppConfig updates config
func UpdateAppConfig(ctx context.Context, cfg *AppConfig) (interface{}, error) {
	return resolver.UpdateAppConfig(ctx, cfg)
}

//SaveStats save stats
func SaveStats(ctx context.Context, stats *common.Stats) error {
	return resolver.SaveStats(ctx, stats)
}

//GetStatsKeys get stats per date
func GetStatsKeys(ctx context.Context, date string) (map[string]int, error) {
	return resolver.GetStatsKeys(ctx, date)
}

//GetStatsKeys get stats per date
func (f FileConfigResolver) GetStatsKeys(ctx context.Context, date string) (map[string]int, error) {
	return nil, nil
}

//GetAppStats get stats
func GetAppStats(ctx context.Context, req *common.StatReq) ([]common.Stats, error) {
	return resolver.GetAppStats(ctx, req)
}

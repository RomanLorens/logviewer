package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"crypto/tls"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/RomanLorens/logviewer/common"
	"github.com/RomanLorens/rl-common/filter"
)

//MongoConfigResolver gets config from file
type MongoConfigResolver struct {
	FilePath string
}

type mongoCreds struct {
	URI              string `json:"uri"`
	DB               string `json:"database"`
	AppsCollection   string `json:"apps_collection"`
	ConfigCollection string `json:"config_collection"`
}

type zeroSource struct{}

func (zeroSource) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

type logWriter struct{}

var (
	//TLSConfigAllTrust trust all connections
	TLSConfigAllTrust = &tls.Config{
		KeyLogWriter:       &logWriter{},
		Rand:               zeroSource{},
		InsecureSkipVerify: true,
	}
	_db *mongo.Database
)

//Write logs message
func (*logWriter) Write(p []byte) (n int, err error) {
	logger.Info(context.Background(), fmt.Sprintf("mongo tls connection %v", string(p)))
	return len(p), nil
}

//SaveStats save stats
func (f MongoConfigResolver) SaveStats(ctx context.Context, stats *common.Stats) error {
	db, err := f.connectDB(ctx)
	if err != nil {
		return fmt.Errorf("Could not connect to db, %v", err)
	}
	stats.CreatedOn = time.Now().String()
	c := db.Collection("stats")
	if er := ctx.Err(); er != nil {
		//TODO why this context was cancelled
		ctx = context.Background()
		logger.Error(ctx, "Context error - %v", er)
	}
	res, err := c.InsertOne(ctx, stats)
	if err != nil {
		return fmt.Errorf("Could not insert stats, %v", err)
	}
	logger.Info(ctx, "Inserted stats %v", res)
	return nil
}

//GetStatsKeys get stats per date
func (f MongoConfigResolver) GetStatsKeys(ctx context.Context, date string) (map[string]int, error) {
	db, err := f.connectDB(ctx)
	if err != nil {
		return nil, err
	}
	logger.Info(ctx, "get stats per %v day", date)
	cur, er := db.Collection("stats").Find(ctx, bson.D{{"date", date}})
	if er != nil {
		return nil, fmt.Errorf("Find stats failed, %v", er)
	}
	defer cur.Close(ctx)
	stats := make(map[string]int, 10)
	for cur.Next(ctx) {
		var s common.Stats
		err := cur.Decode(&s)
		if err != nil {
			return nil, fmt.Errorf("Could not decode from mongo %v", err)
		}
		stats[common.StatsKey(&s)] = 0
	}
	return stats, nil
}

//GetAppStats get app stats
func (f MongoConfigResolver) GetAppStats(ctx context.Context, req *common.StatReq) ([]common.Stats, error) {
	db, err := f.connectDB(ctx)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}
	from := time.Unix(req.From, 0).Format("2006-01-02")
	to := time.Unix(req.To, 0).Format("2006-01-02")
	filter := bson.D{{"app", req.App}, {"env", req.Env}, {"logPath", req.Log}, {"date", bson.D{{"$gte", from}, {"$lte", to}}}}
	logger.Info(ctx, "get app stats %v", filter)
	cur, er := db.Collection("stats").Find(ctx, filter)
	if er != nil {
		return nil, fmt.Errorf("Find stats failed, %v", er)
	}
	defer cur.Close(ctx)
	stats := make([]common.Stats, 0, 10)
	for cur.Next(ctx) {
		var s common.Stats
		err := cur.Decode(&s)
		if err != nil {
			return nil, fmt.Errorf("Could not decode from mongo %v", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

//GetConfig config from mongo
func (f MongoConfigResolver) GetConfig(ctx context.Context) (*Configuration, error) {
	logger.Info(ctx, "Loading mongo config with resolver %v", f)
	res, err := f.doWithMongo(context.Background(), func(client *mongo.Client, creds *mongoCreds) (interface{}, error) {

		appsCollection := client.Database(creds.DB).Collection(creds.AppsCollection)
		cur, err := appsCollection.Find(ctx, bson.M{})
		if err != nil {
			return nil, fmt.Errorf("Error for getting mongo configs with %v creds, %v", creds, err)
		}
		defer cur.Close(ctx)
		cfgs := make([]AppConfig, 0)
		for cur.Next(ctx) {
			var elem AppConfig
			err := cur.Decode(&elem)
			if err != nil {
				return nil, fmt.Errorf("Could not decode from mongo %v", err)
			}
			elem.ID = cur.Current.Lookup("_id").ObjectID().Hex()
			cfgs = append(cfgs, elem)
		}

		configuration := Configuration{ApplicationsConfig: cfgs}
		c := client.Database(creds.DB).Collection(creds.ConfigCollection)
		dbConfig := make(map[string]interface{})
		err = c.FindOne(ctx, bson.M{}).Decode(&dbConfig)
		if err != nil {
			logger.Panicf(ctx, "Could not get data from config db %v", err)
		}
		ips, err := getWhiteListIPs(dbConfig)
		if err != nil {
			logger.Error(ctx, "Could not get whitelist ips, %v", err)
		} else {
			configuration.WhiteListIPs = ips
		}

		v, ok := dbConfig["userByLoginIdUrl"]
		if !ok {
			return nil, fmt.Errorf("Missing mongodb config userByLoginIdUrl")
		}
		configuration.UserByLoginIDURL = v.(string)
		if v, ok = dbConfig["bearer"]; ok {
			configuration.Bearer = v.(string)
		}
		if v, ok = dbConfig["emailServer"]; ok {
			configuration.EmailServer = v.(string)
		} else {
			configuration.EmailServer = "localhost:25"
		}
		emails, er := getStatsEmailReciepients(dbConfig)
		if er != nil {
			logger.Error(ctx, "Could not set stats email reciepients, %v", er)
		} else {
			configuration.StatsEmailReciepients = emails
		}

		logger.Info(ctx, "Loaded %v configs from mongo", len(cfgs))
		return &configuration, nil
	})

	if err != nil {
		logger.Panicf(context.Background(), "Could not connect to mongodb, %v", err)
		return nil, err
	}
	return res.(*Configuration), nil
}

func (f MongoConfigResolver) connectDB(ctx context.Context) (*mongo.Database, error) {
	//todo threads
	if _db != nil {
		return _db, nil
	}
	mongoCfg, er := creds(f.FilePath)
	if er != nil {
		return nil, er
	}
	clientOptions := options.Client().ApplyURI(mongoCfg.URI)
	if !strings.Contains(mongoCfg.URI, "localhost") {
		//TODO should use certs - this trusts all
		clientOptions.SetTLSConfig(TLSConfigAllTrust)
	}
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to db %v", err)
	}
	_db = client.Database(mongoCfg.DB)
	return _db, nil
}

func (f MongoConfigResolver) doWithMongo(ctx context.Context, callback func(c *mongo.Client, creds *mongoCreds) (interface{}, error)) (interface{}, error) {
	mongoCfg, er := creds(f.FilePath)
	if er != nil {
		return nil, er
	}
	clientOptions := options.Client().ApplyURI(mongoCfg.URI)
	if !strings.Contains(mongoCfg.URI, "localhost") {
		//TODO should use certs - this trusts all
		clientOptions.SetTLSConfig(TLSConfigAllTrust)
	}
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to db %v", err)
	}
	defer func(client *mongo.Client) {
		err := client.Disconnect(ctx)
		if err != nil {
			logger.Error(ctx, "Could not close connection %v", err)
		}
		logger.Info(ctx, "Closed mongo connection")
	}(client)

	return callback(client, mongoCfg)
}

func getStatsEmailReciepients(db map[string]interface{}) ([]string, error) {
	emails, ok := db["statsEmailReciepients"]
	if !ok {
		return nil, fmt.Errorf("Missing 'statsEmailReciepients' config")
	}
	arr, ok := emails.(primitive.A)
	if !ok {
		return nil, fmt.Errorf("'statsEmailReciepients' config should be an array")
	}
	e := make([]string, 0, len(arr))
	for _, a := range arr {
		e = append(e, a.(string))
	}
	return e, nil
}

func getWhiteListIPs(db map[string]interface{}) ([]filter.WhiteList, error) {
	ips, ok := db["whitelisted_ips"]
	if !ok {
		return nil, errors.New("Missing whitelisted_ips db config")
	}
	arr, ok := ips.(primitive.A)
	if !ok {
		return nil, fmt.Errorf("Could not cast as slice, %T", arr)
	}

	res := make([]filter.WhiteList, len(arr))
	for _, a := range arr {
		m, ok := a.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("Could not cast as map, %T", a)
		}
		_endpoints, ok := m["endpoints"].(primitive.A)
		if !ok {
			return nil, fmt.Errorf("Could not cast as slice of endpoints, %T", _endpoints)
		}
		endpoints := make([]string, len(_endpoints))
		for _, e := range _endpoints {
			endpoints = append(endpoints, e.(string))
		}
		wip := filter.WhiteList{
			IP:        m["ip"].(string),
			User:      m["user"].(string),
			IsAdmin:   m["isAdmin"].(bool),
			Endpoints: endpoints,
			Token:     m["token"].(string),
		}
		res = append(res, wip)
	}
	return res, nil
}

func creds(path string) (*mongoCreds, error) {
	var creds mongoCreds
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, %v", path, err)
	}
	if err = json.Unmarshal(b, &creds); err != nil {
		return nil, fmt.Errorf("Could not unmarshal config %v", err)
	}
	return &creds, nil
}

//UpdateAppConfig update cfg
func (f MongoConfigResolver) UpdateAppConfig(ctx context.Context, cfg *AppConfig) (interface{}, error) {
	return f.doWithMongo(context.Background(), func(client *mongo.Client, creds *mongoCreds) (interface{}, error) {
		appsCollection := client.Database(creds.DB).Collection(creds.AppsCollection)
		if cfg.ID != "" {
			logger.Info(ctx, "Updating app config %v", cfg.ID)
			_id, err := primitive.ObjectIDFromHex(cfg.ID)
			if err != nil {
				return nil, fmt.Errorf("Could not convert id, %v", err)
			}
			//TODO to prevenet extra field
			cfg.ID = ""
			res, err := appsCollection.UpdateOne(ctx, bson.M{"_id": _id}, bson.M{"$set": cfg})
			if err != nil {
				return nil, fmt.Errorf("Could not update doc, %v", err)
			}
			return res, nil
		}
		logger.Info(ctx, "Creating app config %v", cfg.Application)
		res, err := appsCollection.InsertOne(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("Could not insert new doc, %v", err)
		}
		return res, nil
	})
}

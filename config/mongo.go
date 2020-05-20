package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"crypto/tls"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	e "github.com/RomanLorens/logviewer/error"
	"github.com/RomanLorens/logviewer/logger"
)

//MongoConfigResolver gets config from file
type MongoConfigResolver struct {
	FilePath string
}

type mongoCreds struct {
	URI        string `json:"uri"`
	DB         string `json:"database"`
	Collection string `json:"collection"`
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
)

//Write logs message
func (*logWriter) Write(p []byte) (n int, err error) {
	logger.Info(context.Background(), fmt.Sprintf("mongo tls connection %v", string(p)))
	return len(p), nil
}

//GetConfig config from mongo
func (f MongoConfigResolver) GetConfig() ([]*Config, *e.Error) {
	mongoCfg, er := creds(f.FilePath)
	if er != nil {
		return nil, er
	}
	clientOptions := options.Client().ApplyURI(mongoCfg.URI)
	if !strings.Contains(mongoCfg.URI, "localhost") {
		//TODO should use certs - this trusts all
		clientOptions.SetTLSConfig(TLSConfigAllTrust)
	}
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		logger.Panicf(context.Background(), "Could not connect to db %v", err)
	}
	defer func(client *mongo.Client) {
		err := client.Disconnect(context.Background())
		if err != nil {
			logger.Error(context.Background(), "Could not close connection %v", err)
		}
		logger.Info(context.Background(), "Closed mongo connection")
	}(client)

	collection := client.Database(mongoCfg.DB).Collection(mongoCfg.Collection)
	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, e.Errorf(http.StatusBadRequest, "Error for getting mongo configs with %v creds, %v", creds, err)
	}
	defer cur.Close(context.Background())
	cfgs := make([]*Config, 0)
	for cur.Next(context.Background()) {
		var elem Config
		err := cur.Decode(&elem)
		if err != nil {
			return nil, e.Errorf(http.StatusBadRequest, "Could not decode from mongo %v", err)
		}
		cfgs = append(cfgs, &elem)
	}
	logger.Info(context.Background(), "Loaded %v configs from mongo", len(cfgs))
	return cfgs, nil
}

func creds(path string) (*mongoCreds, *e.Error) {
	var creds mongoCreds
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not read file %s, %v", path, err)
	}
	if err = json.Unmarshal(b, &creds); err != nil {
		return nil, e.Errorf(http.StatusInternalServerError, "Could not unmarshal config %v", err)
	}
	return &creds, nil

}

#!/bin/bash

go run main.go -config=mongo -configFile=static/mongo-sit.json -enableScheduler=false -cert=static/https-server.crt -certKey=static/https-server.key
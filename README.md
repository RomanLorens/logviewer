# logviewer
Backend for logviewer - allows to view remote logs for various applications,
There should be passed configuration with monitored hosts, currently configuration may be fetched from static file or mongodb, eg

go run main.go -config=mongo -configFile=mongo.json

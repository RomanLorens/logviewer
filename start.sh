#!/bin/bash

count=`ps -ef | grep logviewer | grep -v grep | wc -l`
if (( $count == 0 )); then
	echo "No logviewer instance running, will restart..."	
	nohup ./logviewer -config=mongo -configFile=cfg/mongo-sit.json -staticFolder=./logviewer-ui -enableScheduler=true -cert=cfg/https-server.crt -certKey=cfg/https-server.key >/dev/null 2>&1 &

else
	echo "logviewer instance already runnig"
fi

exit 0

#!/bin/bash

killall shorturl-api
killall shorturl-web

docker stop mongo1
docker stop memcache1

docker run --rm --name mongo1 -d -v /home/tekieli/workspace/mongo_db_data/:/data/db mongo:latest
docker run --rm --name memcache1 -d memcached:latest memcached -m 64 -vvv

./shorturl-api --db-ip=192.168.30.2 --cache-ip=192.168.30.3 --port=8090 &
./shorturl-web --api-server=192.168.30.1:8090 --port=8080 --static=./web &

sleep 5

curl -v --data '{"url": "http://example.com"}' localhost:8080/api/shorten

echo "Press any key to continue"
while [ true ] ; do
    read -t 3 -n 1
    if [ $? = 0 ] ; then
        break ;
    else
        echo "waiting for the keypress"
    fi
done

killall shorturl-api
killall shorturl-web

docker stop mongo1
docker stop memcache1

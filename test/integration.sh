#!/bin/bash

docker run --rm --name mongo1 -d -v /home/tekieli/workspace/mongo_db_data/:/data/db mongo:latest
docker run --rm --name memcache1 -d memcached:latest memcached -m 64 -vvv

./shorturl --db-ip=192.168.30.2 --cache-ip=192.168.30.3 &

sleep 5

curl -v --data '{"url": "http://example.com"}' localhost:8080/api/shorten

killall shorturl

docker stop mongo1
docker stop memcache1

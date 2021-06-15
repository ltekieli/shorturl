#!/bin/bash

docker stop shorturl-api1
docker stop shorturl-web1
docker stop shorturl-mongo1
docker stop shorturl-memcached1

docker volume create shorturl-mongo1-data

docker run \
    --rm \
    --name shorturl-mongo1 \
    -d \
    -v shorturl-mongo1-data:/data/db \
    mongo:latest

docker run \
    --rm \
    --name shorturl-memcached1 \
    -d \
    memcached:latest \
    memcached -m 64

docker run \
    --rm \
    --name shorturl-api1 \
    -d \
    -p 8090:8090 \
    shorturl-api:latest \
    shorturl-api --db-ip=192.168.30.2 --cache-ip=192.168.30.3 --port=8090

docker run \
    --rm \
    --name shorturl-web1 \
    -d \
    -p 8080:8080 \
    shorturl-web:latest \
    shorturl-web --api-server=192.168.30.4:8090 --port=8080 --static=/var/www

sleep 5

curl -v --data '{"url": "http://example.com"}' localhost:8090/api/shorten

echo "Press any key to continue"
while true ; do
    if read -r -t 3 -n 1 ; then
        break ;
    else
        echo "waiting for the keypress"
    fi
done

docker stop shorturl-api1
docker stop shorturl-web1
docker stop shorturl-mongo1
docker stop shorturl-memcached1

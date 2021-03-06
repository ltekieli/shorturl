#!/bin/bash

docker-compose -f test/docker-compose.yml down
docker-compose -f test/docker-compose.yml up -d --scale shorturl-web=3 --scale shorturl-api=3

sleep 5

curl -v --data '{"url": "http://example.com"}' http://localhost/api/shorten

echo "Press any key to continue"
while true ; do
    if read -r -t 3 -n 1 ; then
        break ;
    else
        echo "waiting for the keypress"
    fi
done

docker-compose -f test/docker-compose.yml down

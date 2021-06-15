#!/bin/bash

docker build -f build/dockerfile-api -t shorturl-api .
docker build -f build/dockerfile-web -t shorturl-web .

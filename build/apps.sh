#!/bin/bash

set -euo pipefail

go build -v -o shorturl-api cmd/server_api/main.go
go build -v -o shorturl-web cmd/server_web/main.go

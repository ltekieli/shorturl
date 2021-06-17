#!/bin/bash

set -euo pipefail

./build/apps.sh
./build/docker.sh

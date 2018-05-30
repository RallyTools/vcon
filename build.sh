#!/bin/bash

set -euo pipefail

if [[ "${*-}" == *release* ]]; then
	go build -o ./bin/vcon -ldflags '-s -w' ./main
else
	go build -o ./bin/vcon ./main
fi

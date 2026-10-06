#!/usr/bin/env bash

set -euo pipefail

mkdir -p bin

go build \
  -trimpath \
  -o bin/lab \
  ./cmd/lab

echo
echo "Built: bin/lab"
./bin/lab version
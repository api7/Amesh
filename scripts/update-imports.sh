#!/bin/bash

set -e

PROJECT_NAME=github.com/api7/amesh
while IFS= read -r -d '' file; do
  goimports-reviser  -file-path "$file" -project-name $PROJECT_NAME
done <   <(find . -name '*.go' -not -path "./test/*" -not -path "./controller/apis/client/**" -not -path './controller/main.go' -not -path './api/go' -print0)

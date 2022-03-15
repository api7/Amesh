#!/bin/bash

set -e

PROJECT_NAME=github.com/api7/amesh
while IFS= read -r -d '' file; do
  goimports-reviser  -file-path "$file" -project-name $PROJECT_NAME
done <   <(find . -name '*.go' -not -path "./test/*" -print0)

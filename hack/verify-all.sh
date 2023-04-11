#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
make -C "${SCRIPT_ROOT}" tidy
STATUS=$(cd "${SCRIPT_ROOT}" && git status --porcelain go.mod go.sum)
if [ ! -z "$STATUS" ]; then
  git diff --color go.mod go.sum | cat
  echo "Running 'go mod tidy' to fix your 'go.mod' and/or 'go.sum'"
  exit 1
fi
echo "go module is tidy."

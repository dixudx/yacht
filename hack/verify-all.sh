#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
EXAMPLES=$(cd "${SCRIPT_ROOT}"/examples; ls -d * | paste -s -d, -)

check-go-module() {
  echo "Checking go modules in folder ${1}"
  go mod tidy
  STATUS=$(cd "${1}" && git status --porcelain go.mod go.sum)
  if [ ! -z "$STATUS" ]; then
    git diff --color go.mod go.sum | cat
    echo "Running 'go mod tidy' to fix your 'go.mod' and/or 'go.sum'"
    exit 1
  fi
  echo "go module is tidy."
}

check-go-module "${SCRIPT_ROOT}"

IFS="," read -ra examples <<<"${EXAMPLES}"
for example in "${examples[@]}"; do
  check-go-module "${SCRIPT_ROOT}/examples/${example}"
done

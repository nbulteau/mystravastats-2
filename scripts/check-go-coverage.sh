#!/usr/bin/env sh

set -eu

repository_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
coverage_file=${GO_COVERAGE_FILE:-"$repository_dir/back-go/coverage.out"}
minimum_coverage=${GO_COVERAGE_MINIMUM:-50.0}

cd "$repository_dir/back-go"
go test -covermode=atomic -coverprofile="$coverage_file" ./...

total_coverage=$(go tool cover -func="$coverage_file" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')
if [ -z "$total_coverage" ]; then
    echo "Unable to read total Go coverage." >&2
    exit 1
fi

awk -v actual="$total_coverage" -v minimum="$minimum_coverage" 'BEGIN {
    if ((actual + 0) < (minimum + 0)) {
        printf "Go coverage %.1f%% is below the required %.1f%%.\n", actual, minimum > "/dev/stderr"
        exit 1
    }
    printf "Go coverage %.1f%% meets the required %.1f%%.\n", actual, minimum
}'

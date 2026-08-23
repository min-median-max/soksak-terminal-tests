#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
stage=${1:-$root/local/windows-preflight}
case "$stage" in /*) ;; *) stage="$root/$stage" ;; esac

[ "$(go version | awk '{print $3}')" = "go1.26.3" ] || { echo "Go 1.26.3 is required" >&2; exit 1; }
mkdir -p "$stage"

go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 go test -c -tags=system -o "$stage/windows-system.exe" ./system
go run ./cmd/stage-fleet -platform windows -target x86_64-pc-windows-msvc -stage "$stage"
test -s "$stage/windows-system.exe"
go version -m "$stage/windows-system.exe" | grep -F 'GOOS=windows' >/dev/null
test -s "$stage/composition-home/settings.json"
test -s "$stage/composition-home/installed.json"

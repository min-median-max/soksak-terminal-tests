#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
stage=${1:-$root/local/windows-preflight}
case "$stage" in /*) ;; *) stage="$root/$stage" ;; esac

required=$(awk '$1 == "go" { print "go" $2; count++ } END { if (count != 1) exit 1 }' go.mod)
actual=$(go env GOVERSION)
[ "$actual" = "$required" ] || { echo "$required is required; found $actual" >&2; exit 1; }
mkdir -p "$stage"

go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 go test -c -tags=system -o "$stage/windows-system.exe" ./system
go run ./cmd/verify-fleet -platform windows -target x86_64-pc-windows-msvc
test -s "$stage/windows-system.exe"
go version -m "$stage/windows-system.exe" | grep -F 'GOOS=windows' >/dev/null

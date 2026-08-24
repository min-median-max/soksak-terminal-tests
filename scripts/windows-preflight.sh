#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
stage=${1:-$root/local/windows-preflight}
case "$stage" in /*) ;; *) stage="$root/$stage" ;; esac

mkdir -p "$stage"

make verify
GOOS=windows GOARCH=amd64 go test -c -tags=system -o "$stage/windows-system.exe" ./system
make fleet TARGET=x86_64-pc-windows-msvc
test -s "$stage/windows-system.exe"
go version -m "$stage/windows-system.exe" | grep -F 'GOOS=windows' >/dev/null

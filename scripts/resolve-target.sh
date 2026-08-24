#!/bin/sh
set -eu

[ "$#" -eq 1 ] && [ -n "$1" ] || { echo 'usage: resolve-target.sh <target>' >&2; exit 2; }
case "$1" in
  aarch64-apple-darwin) printf 'darwin arm64\n' ;;
  x86_64-apple-darwin) printf 'darwin amd64\n' ;;
  aarch64-unknown-linux-gnu) printf 'linux arm64\n' ;;
  x86_64-unknown-linux-gnu) printf 'linux amd64\n' ;;
  x86_64-pc-windows-msvc) printf 'windows amd64\n' ;;
  *) echo "unsupported acceptance target: $1" >&2; exit 2 ;;
esac

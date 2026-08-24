#!/bin/sh
set -eu

[ "$#" -eq 1 ] && [ -n "$1" ] || { echo 'usage: check-native-target.sh <target>' >&2; exit 78; }
target=$1
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
set -- $("$root/scripts/resolve-target.sh" "$target")
target_os=$1
target_arch=$2
go_host_os=$(go env GOHOSTOS 2>/dev/null || true)
go_host_arch=$(go env GOHOSTARCH 2>/dev/null || true)
if [ "$target_os" != "$go_host_os" ] || [ "$target_arch" != "$go_host_arch" ]; then
  printf 'TOOLCHAIN_MISMATCH: native target=%s requires=%s/%s actualRuntime=%s/%s\n' \
    "$target" "$target_os" "$target_arch" "${go_host_os:-unknown}" "${go_host_arch:-unknown}" >&2
  exit 78
fi
printf 'NATIVE_TARGET_READY target=%s runtime=%s/%s\n' "$target" "$go_host_os" "$go_host_arch"

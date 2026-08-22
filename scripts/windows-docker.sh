#!/bin/sh
set -eu

tests=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
core=${SOKSAK_CORE:-$tests/../../soksak-core}
core=$(CDPATH= cd -- "$core" && pwd)

"$core/scripts/ci/windows-docker.sh"

docker run --rm -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
  -v "$tests:/tests" -v soksak-windows-tests-go-mod:/go/pkg/mod -v soksak-windows-tests-go-build:/tmp/go-build \
  -e GOCACHE=/tmp/go-build --entrypoint /bin/sh soksak-windows-ci:node24-wails-beta12 \
  -c 'mkdir -p /tmp/home /tmp/go-build /go/pkg/mod && chown -R "$HOST_UID:$HOST_GID" /tmp/home /tmp/go-build /go/pkg/mod && exec setpriv --reuid="$HOST_UID" --regid="$HOST_GID" --clear-groups env HOME=/tmp/home GOCACHE=/tmp/go-build /bin/sh -c "cd /tests && scripts/windows-preflight.sh local/windows-preflight"'

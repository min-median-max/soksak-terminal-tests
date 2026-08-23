#!/bin/sh
set -eu

: "${SOKSAK_TEST_SETTINGS:?}"
: "${SOKSAK_TEST_CLI:?}"
: "${SOKSAK_TEST_APP:?}"
: "${SOKSAK_TEST_HOME:?}"
: "${SOKSAK_TEST_RUNTIME:?}"
: "${SOKSAK_TEST_WORKSPACE:?}"
: "${SOKSAK_TEST_IDENTIFIER:?}"
: "${SOKSAK_TEST_EVIDENCE:?}"
case "$SOKSAK_TEST_RUNTIME" in /tmp/*|/private/tmp/*) ;; *) echo "Darwin system runtime must use a short temporary path" >&2; exit 1 ;; esac
exec go test -count=1 -tags=system ./system "$@"

#!/bin/sh
set -eu

: "${SOKSAK_TEST_CLI:?}"
: "${SOKSAK_TEST_APP:?}"
: "${SOKSAK_TEST_HOME:?}"
: "${SOKSAK_TEST_RUNTIME:?}"
: "${SOKSAK_TEST_WORKSPACE:?}"
: "${SOKSAK_TEST_IDENTIFIER:?}"
: "${SOKSAK_TEST_EVIDENCE:?}"
test -x /bin/bash
export SHELL=/bin/bash

dbus-run-session -- xvfb-run -a -s "-screen 0 1920x1200x24" sh -eu -c '
  printf "\n" | gnome-keyring-daemon --unlock >/tmp/soksak-keyring.env
  wm_ready=/tmp/soksak-openbox-ready.$$
  rm -f "$wm_ready"
  openbox --startup "touch $wm_ready" >/tmp/soksak-openbox.log 2>&1 &
  while [ ! -f "$wm_ready" ]; do inotifywait -qq -e create /tmp; done
  rm -f "$wm_ready"
  export GDK_BACKEND=x11
  export GSK_RENDERER=cairo
  export GTK_A11Y=none
  export NO_AT_BRIDGE=1
  export LIBGL_ALWAYS_SOFTWARE=1
  export WEBKIT_DISABLE_COMPOSITING_MODE=1
  export WEBKIT_DISABLE_DMABUF_RENDERER=1
  export WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1
  exec go test -count=1 -tags=system ./system "$@"
' sh "$@"

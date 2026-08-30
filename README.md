# soksak-terminal-tests

Non-product quality repository for terminal implementations used by Soksak.

- benchmark compares provider throughput, RSS and loss against measured daemon demand.
- system installs released components through Soksak commands and runs black-box recovery scenarios.

This repository is not installable software and is never listed in soksak-plugin-registry. It
contains no plugin, sidecar or kit implementation. It does not build another repository's source
tree. Inputs are installed artifacts, public commands, contract reports and benchmark interfaces.

Each plugin and sidecar repository remains responsible for its own conformance and release gates.
This repository owns only cross-provider comparison and installed-system behavior.

Darwin and Linux require seven plugins plus PTY and six recovery sidecars. Windows requires the five
plugins backed by Alacritty, Ghostty, VT100, WezTerm, and Xterm plus PTY and four recovery sidecars;
Kitty and Shitty are not Windows artifacts. `verify-fleet` validates those immutable release bytes.
The system suite starts Core with an empty home and calls `plugin.install` with only the authenticated
Registry plugin identity. Each plugin release declares its exact Sidecar closure, which Core installs.
The suite reads every consent summary, grants consent, enables each plugin, and validates
the resulting `environment.json` through the public control surface. This repository never writes
product installation state.

Provider repositories produce versioned `*.bench.json` owner reports containing feed,
serialization and memory costs only. This repository reads those reports and never runs provider
source. Real PTY demand, recovery gaps, tail delivery and attached end-to-end throughput belong to
the installed system scenarios here:

```sh
SOKSAK_BENCH_REPORTS=/absolute/path/to/reports make benchmark
```

The default test suite validates the harness. Native system tests require the application and CLI:

```sh
SOKSAK_TEST_PLATFORM=linux \
SOKSAK_TEST_CLI=/absolute/path/to/sok \
SOKSAK_TEST_APP=/absolute/path/to/soksak \
SOKSAK_TEST_RUNTIME=/absolute/path/to/runtime \
SOKSAK_TEST_WORKSPACE=/absolute/path/to/workspace \
SOKSAK_TEST_IDENTIFIER=com.soksak.systemtest \
SOKSAK_TEST_EVIDENCE=/absolute/path/to/evidence \
make system-commands TARGET=x86_64-unknown-linux-gnu
```

Run `make system-restore TARGET=<native-target>` for the restore scenario and `make verify` for this
repository's owner tests. `go.mod` is the only Go-version owner. `fleet TARGET=<target>` may audit an
immutable artifact fleet from another host, but `system-*` additionally requires the target to match
the actual host runtime. A cross-compiled test binary is never accepted as native runtime evidence.

`make system-native-focus`, `make system-native-cursor`, and `make system-native-keyboard` with
`TARGET=aarch64-apple-darwin` are the ordered Darwin AppKit certification gates. They deliberately
activate their isolated test application because WebKit does not deliver
native key events to an inactive, non-key window. Run it only on an unattended native runner; local
capture-only development uses `system-theme` and `system-performance` and must not take the user's focus. The gate sends
AppKit mouse and key events to public-node coordinates and proves the route reaches each terminal
and PTY. It consumes `SOKSAK_TEST_CANDIDATE_PLAN` when supplied and otherwise installs the declared
immutable fleet through Core.

Every terminal resize writes `<plugin-id>-resize.json` under `SOKSAK_TEST_EVIDENCE`. The record
contains the pane resize receipt, the exposed terminal root's wide and narrow CSS-pixel rectangles,
the requested PTY size, PTY observation and absolute output sequence, recovery observation, output
sequence and gap count, and the renderer's applied output sequence. High-output checks also write
`<plugin-id>-output.json`. Failure names the first boundary that did not advance. A plain timeout
without these coordinates is not a verdict.

Screenshots and recordings are mandatory visual-review artifacts, but their pixels are not an
automated pass/fail oracle. Colour parity is decided from the public `terminal-screen` computed
foreground/background, cursor and selection properties, plus all 256 contract-owned ANSI custom
properties. Capture generation failures are harness failures; a person still inspects the emitted
images before completion is claimed.

The candidate workflow runs `system-theme → system-native-focus → system-native-cursor →
system-native-keyboard → system-visibility →
system-performance → system-commands`. Theme and native input are therefore not blocked by a later
performance RED.

## Installed Vision six-engine matrix

`verify-installed-terminal-matrix` checks an already running installed product without installing
components or reading any product source. The caller declares the absolute CLI and optional socket,
workspace window, host pane, expected fresh prompt marker, existing non-symlink snapshot directory,
and the exact version plus executable SHA-256 for all six providers:

```sh
go run ./cmd/verify-installed-terminal-matrix \
  --cli /absolute/path/to/sok \
  --socket /absolute/path/to/control.sock \
  --window w-installed \
  --pane g-host \
  --prompt-marker 'FRESH_PROMPT_MARKER' \
  --snapshot-dir /absolute/path/to/matrix-images \
  --expect "alacritty=0.0.42@${ALACRITTY_SHA256}" \
  --expect "ghostty=0.0.40@${GHOSTTY_SHA256}" \
  --expect "kitty=0.0.36@${KITTY_SHA256}" \
  --expect "shitty=0.0.37@${SHITTY_SHA256}" \
  --expect "vt100=0.0.39@${VT100_SHA256}" \
  --expect "wezterm=0.0.38@${WEZTERM_SHA256}"
```

Each `*_SHA256` variable must be declared as the expected 64-character lowercase executable
digest before running the command. The snapshot directory must already exist.

Each engine transaction uses only public sok commands: select Vision's engine, open a fresh
`terminal-vision` tab, use the event-backed live wait, validate the addressed view/pane and zero
recovery gaps, perform an unscoped full read for the prompt marker, and verify the exact four-field
`sidecar.status` identity against the declared executable digest. It then requires settled layout
and zero composition violations before requesting `window.snapshot` for that tab without activating
or focusing it. The JSON report names every snapshot for human inspection. There are no polling
loops, sleeps, Enter-key workarounds, PATH changes, or provider-specific exceptions.

## Verification identity

A GREEN result belongs only to one immutable input set: Core commit, this repository commit,
Registry asset digest, every selected release.json digest, and OS/architecture. Changing any member
invalidates the prior result and requires the canonical local gate again. If the same set produces
different results, that is an idempotence or observation defect; rerunning until success is not a
valid result. Provider exceptions, timeout increases, and retry loops cannot satisfy this rule.

System commands are generated for the declared target shell. Windows uses `cmd.exe` commands and
UTF-16LE PowerShell `-EncodedCommand` payloads; Darwin and Linux use POSIX shell syntax. Restore
markers are scheduled without placing the awaited marker literal in the echoed command, and the
test reads PTY process identity only from the public `pty.status` contract.

The macOS native runner executes the universal application on Apple Silicon with the full seven
plugin fleet. Its control socket uses a short temporary path because Darwin limits Unix socket
addresses. The Windows native runner alone decides WebView2, ConPTY, and named-pipe behavior.

Inside the Linux system-test image, run the same suite through the D-Bus, Secret Service and Xvfb
runner. The runner declares `/bin/bash` as the login shell. All `SOKSAK_TEST_*` paths must name container paths. Additional arguments are passed to
`go test`.

```sh
make system-commands TARGET=x86_64-unknown-linux-gnu
```

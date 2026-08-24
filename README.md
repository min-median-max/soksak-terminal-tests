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

`make system-native-input TARGET=aarch64-apple-darwin` is the Darwin AppKit input certification
gate. It deliberately activates its isolated test application because WebKit does not deliver
native key events to an inactive, non-key window. Run it only on an unattended native runner; local
capture-only development must use `system-parity` and must not take the user's focus. The gate sends
AppKit mouse and key events to public-node coordinates and proves the route reaches each terminal
and PTY. It consumes `SOKSAK_TEST_CANDIDATE_PLAN` when supplied and otherwise installs the declared
immutable fleet through Core.

Every terminal resize writes `<plugin-id>-resize.json` under `SOKSAK_TEST_EVIDENCE`. The record
contains the pane resize receipt, the exposed terminal root's wide and narrow CSS-pixel rectangles,
the requested PTY size, PTY observation and absolute output sequence, recovery observation, output
sequence and gap count, and the renderer's applied output sequence. High-output checks also write
`<plugin-id>-output.json`. Failure names the first boundary that did not advance. A plain timeout
without these coordinates is not a verdict.

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

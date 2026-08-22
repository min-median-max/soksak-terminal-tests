# soksak-terminal-tests

Non-product quality repository for terminal implementations used by Soksak.

- benchmark compares provider throughput, RSS and loss against measured daemon demand.
- system runs black-box recovery scenarios against installed Soksak settings.

This repository is not installable software and is never listed in soksak-plugin-registry. It
contains no plugin, sidecar or kit implementation. It does not build another repository's source
tree. Inputs are installed artifacts, public commands, contract reports and benchmark interfaces.

Each plugin and sidecar repository remains responsible for its own conformance and release gates.
This repository owns only cross-provider comparison and installed-system behavior.

`prepare-development` accepts one JSON object on stdin and atomically writes a development
`settings.json` and `installed.json`. The input declares `platform` (`darwin`, `linux`, or `windows`),
an absolute identity-home path, and exact artifact path, repository, source commit and digest records.
Darwin and Linux require seven plugins plus PTY and six recovery sidecars. Windows requires the five
plugins backed by Alacritty, Ghostty, VT100, WezTerm, and Xterm plus PTY and four recovery sidecars;
Kitty and Shitty are not Windows artifacts.
It validates every manifest identity, process, target and provenance before replacing both
canonical files while the test application is stopped. It does not build source code.

```sh
go run ./cmd/prepare-development < /absolute/path/to/development-input.json
```

Provider repositories produce versioned `*.bench.json` reports. This repository reads those
reports and never runs provider source:

```sh
SOKSAK_BENCH_REPORTS=/absolute/path/to/reports go test -tags=benchmark ./benchmark -v
```

The default test suite validates the harness. The installed inventory gate is explicit and never
skips missing input:

```sh
SOKSAK_TEST_SETTINGS=/absolute/identity-home/settings.json go test -tags=system ./system
```

The public-command gate also requires a running installed application:

```sh
SOKSAK_TEST_SETTINGS=/absolute/identity-home/settings.json \
SOKSAK_TEST_PLATFORM=linux \
SOKSAK_TEST_CLI=/absolute/path/to/sok \
SOKSAK_TEST_APP=/absolute/path/to/soksak \
SOKSAK_TEST_SOCKET=/absolute/path/to/control.sock \
SOKSAK_TEST_HOME=/absolute/path/to/identity-home \
SOKSAK_TEST_RUNTIME=/absolute/path/to/runtime \
SOKSAK_TEST_WORKSPACE=/absolute/path/to/workspace \
SOKSAK_TEST_IDENTIFIER=com.soksak.systemtest \
SOKSAK_TEST_WINDOW=w-workspace \
SOKSAK_TEST_EVIDENCE=/absolute/path/to/evidence \
go test -tags=system ./system
```

Every terminal resize writes `<plugin-id>-resize.json` under `SOKSAK_TEST_EVIDENCE`. The record
contains the pane resize receipt, the exposed terminal root's wide and narrow CSS-pixel rectangles,
the requested PTY size, PTY observation and event sequence, recovery observation and event
sequence, and rendered frame size. Failure names the first boundary that did not advance. A plain
timeout without this record is not a verdict.

System commands are generated for the declared target shell. Windows uses `cmd.exe` commands and
UTF-16LE PowerShell `-EncodedCommand` payloads; Darwin and Linux use POSIX shell syntax. Restore
markers are scheduled without placing the awaited marker literal in the echoed command, and the
test reads PTY process identity only from the public `pty.status` contract.

The macOS Docker runner verifies Windows cross-build outputs and immutable release inputs. It does
not run WebView2, ConPTY or Windows named pipes. Those runtime boundaries are decided only by the
installed system suite on GitHub's `windows-2025` runner.

Inside the Linux system-test image, run the same suite through the D-Bus, Secret Service and Xvfb
runner. The runner declares `/bin/bash` as the login shell. All `SOKSAK_TEST_*` paths must name container paths. Additional arguments are passed to
`go test`.

```sh
scripts/run-linux-system.sh -run TestInstalledTerminalCommands -v
```

# soksak-terminal-tests

Non-product quality repository for terminal implementations used by Soksak.

- benchmark compares provider throughput, RSS and loss against measured daemon demand.
- system runs black-box recovery scenarios against an already installed Soksak settings
  composition.

This repository is not installable software and is never listed in soksak-plugin-registry. It
contains no plugin, sidecar or kit implementation. It does not build another repository's source
tree. Inputs are installed artifacts, public commands, contract reports and benchmark interfaces.

Each plugin and sidecar repository remains responsible for its own conformance and release gates.
This repository owns only cross-provider comparison and installed-composition system behavior.

`prepare-development` accepts one JSON object on stdin and atomically writes a development
`settings.json`. The input contains an absolute identity-home path plus exact maps for seven plugin
source directories and seven staged sidecar directories. It validates every manifest identity and
sidecar process before replacing the file. It does not build source code.

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

Inside the Linux system-test image, run the same suite through the D-Bus, Secret Service and Xvfb
runner. The runner declares `/bin/bash` as the login shell. All `SOKSAK_TEST_*` paths must name container paths. Additional arguments are passed to
`go test`.

```sh
scripts/run-linux-system.sh -run TestInstalledTerminalCommands -v
```

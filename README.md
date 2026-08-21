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

The default test suite validates the harness. The installed inventory gate is explicit and never
skips missing input:

```sh
SOKSAK_TEST_SETTINGS=/absolute/identity-home/settings.json go test -tags=system ./system
```

The public-command gate also requires a running installed application:

```sh
SOKSAK_TEST_SETTINGS=/absolute/identity-home/settings.json \
SOKSAK_TEST_CLI=/absolute/path/to/sok \
SOKSAK_TEST_SOCKET=/absolute/path/to/control.sock \
SOKSAK_TEST_WINDOW=w-workspace \
go test -tags=system ./system
```

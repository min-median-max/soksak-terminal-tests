SHELL := /bin/sh

.PHONY: require-target preflight native-preflight prepare verify benchmark fleet system system-commands system-restore system-parity system-visibility

require-target:
	@test '$(origin TARGET)' = 'command line' && test -n '$(TARGET)' || { echo 'TARGET must be an explicit Make command-line variable' >&2; exit 2; }

preflight:
	@scripts/check-build-environment.sh

native-preflight: require-target preflight
	@scripts/check-native-target.sh '$(TARGET)'

prepare: preflight
	@GOFLAGS=-mod=readonly go mod download
	@go mod verify

verify: prepare
	@go mod tidy -diff
	@go test -count=1 ./...

benchmark: prepare
	@test -n "$$SOKSAK_BENCH_REPORTS" || { echo 'SOKSAK_BENCH_REPORTS must name the owner-report directory' >&2; exit 2; }
	@go test -count=1 -tags=benchmark ./benchmark -v

fleet: require-target prepare
	@set -- $$(scripts/resolve-target.sh '$(TARGET)'); \
		go run ./cmd/verify-fleet -platform "$$1" -target '$(TARGET)'

system-commands: TEST_NAME := TestInstalledTerminalCommands
system-commands: system

system-restore: TEST_NAME := TestInstalledTerminalWarmAndArchivedRestore
system-restore: system

system-parity: TEST_NAME := TestInstalledTerminalPresentationParity
system-parity: system

system-visibility: TEST_NAME := TestInstalledTerminalVisibilityMatrix
system-visibility: system

system: native-preflight prepare
	@test -n '$(TEST_NAME)' || { echo 'system scenario must be one declared Make target' >&2; exit 2; }
	@set -- $$(scripts/resolve-target.sh '$(TARGET)'); \
		case "$$1" in \
			darwin) scripts/run-darwin-system.sh -run '^$(TEST_NAME)$$' -v ;; \
			linux) scripts/run-linux-system.sh -run '^$(TEST_NAME)$$' -v ;; \
			windows) go test -count=1 -tags=system ./system -run '^$(TEST_NAME)$$' -v ;; \
			*) echo "unsupported system platform: $$1" >&2; exit 2 ;; \
		esac

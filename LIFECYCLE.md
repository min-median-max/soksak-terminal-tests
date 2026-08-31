# System-test lifecycle

System tests distinguish two termination operations.

- `Shutdown` requests normal application shutdown. The product may keep restartable Sidecar
  processes running.
- `Finish` terminates test resources. It closes Plugin tabs, disables enabled Plugins, and calls
  `sidecar_stop` for every Sidecar returned by `sidecar_status`.

`Finish` requires each stop response to contain `running:false`. It then requires both `open` and
`recorded` in `sidecar_status` to be empty and verifies that every PID from the initial status no
longer exists before it shuts down the application. The PID check runs once; `sidecar_stop` already
waits for the operating-system process-exit notification. System tests must call `Finish` when they
complete.

The lifecycle writes the result to `process-window-ownership.json` in the declared evidence
directory.

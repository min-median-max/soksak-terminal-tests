package system

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LifecycleConfig struct {
	App          string
	CLI          string
	Socket       string
	Home         string
	Runtime      string
	Workspace    string
	EvidenceDir  string
	Identifier   string
	Presentation string
	Focus        bool
}

type Lifecycle struct {
	config LifecycleConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	log    *os.File
	window string
}

const testControlReadyEvent = "soksak.host.ready"

func NewLifecycle(config LifecycleConfig) (*Lifecycle, error) {
	for name, path := range map[string]string{
		"app": config.App, "CLI": config.CLI, "socket": config.Socket, "home": config.Home,
		"runtime": config.Runtime, "workspace": config.Workspace, "evidence": config.EvidenceDir,
	} {
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("%s path must be absolute: %s", name, path)
		}
	}
	if config.Identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}
	if config.Presentation == "" {
		config.Presentation = "capture-only"
	}
	if config.Presentation != "capture-only" && config.Presentation != "interactive" {
		return nil, fmt.Errorf("presentation must be capture-only or interactive: %s", config.Presentation)
	}
	if err := validateControlAddress(config.Socket); err != nil {
		return nil, err
	}
	for _, path := range []string{config.Home, config.Runtime, config.Workspace, config.EvidenceDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, err
		}
	}
	return &Lifecycle{config: config}, nil
}

func (lifecycle *Lifecycle) Start() error {
	if lifecycle.cmd != nil {
		return fmt.Errorf("application is already running")
	}
	log, err := os.OpenFile(filepath.Join(lifecycle.config.EvidenceDir, "application.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command(lifecycle.config.App)
	cmd.Env = append(os.Environ(),
		"SOKSAK_HOME="+lifecycle.config.Home,
		"SOKSAK_IDENTIFIER="+lifecycle.config.Identifier,
		"SOKSAK_RUNTIME="+lifecycle.config.Runtime,
		"SOKSAK_PRESENTATION="+lifecycle.config.Presentation,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = log.Close()
		return err
	}
	ready := newControlReadyWriter()
	cmd.Stdout, cmd.Stderr = io.MultiWriter(log, ready), log
	if err := cmd.Start(); err != nil {
		_ = log.Close()
		return err
	}
	lifecycle.cmd, lifecycle.stdin, lifecycle.log = cmd, stdin, log
	expectedWindow := lifecycle.window
	var announcement controlReadyEvent
	select {
	case announcement = <-ready.events:
	case <-time.After(45 * time.Second):
		return fmt.Errorf("application did not announce control readiness within 45 seconds: %s", lifecycle.lastLog())
	}
	if announcement.Protocol != 1 || announcement.Socket != lifecycle.config.Socket ||
		announcement.Identifier != lifecycle.config.Identifier || announcement.PID != cmd.Process.Pid {
		return fmt.Errorf("control readiness does not own this run: %+v", announcement)
	}
	targetWindow := expectedWindow
	if targetWindow == "" {
		targetWindow = "main"
	}
	if _, err := lifecycle.client("main").Call("window_renderer_wait", map[string]any{
		"targetWindow": targetWindow, "timeoutMs": 45000,
	}); err != nil {
		return fmt.Errorf("wait for renderer %s: %w: %s", targetWindow, err, lifecycle.lastLog())
	}
	lifecycle.window = targetWindow
	if _, err := lifecycle.client(targetWindow).Call("app.boot.wait", map[string]any{"timeoutMs": 45000}); err != nil {
		return fmt.Errorf("wait for application %s: %w: %s", targetWindow, err, lifecycle.lastLog())
	}
	if expectedWindow != "" {
		return lifecycle.AwaitWindow()
	}
	opened, err := lifecycle.client("main").Call("window.open", map[string]any{
		"root": lifecycle.config.Workspace, "focus": lifecycle.config.Focus,
	})
	if err != nil {
		return err
	}
	window, _ := opened["label"].(string)
	if window == "" {
		return fmt.Errorf("window.open returned no label")
	}
	lifecycle.window = window
	if _, err := lifecycle.client(window).Call("window_renderer_wait", map[string]any{
		"targetWindow": window, "timeoutMs": 45000,
	}); err != nil {
		return err
	}
	return lifecycle.AwaitWindow()
}

func (lifecycle *Lifecycle) OpenWorkspace() (string, error) {
	if lifecycle.window != "" && lifecycle.window != "main" {
		return lifecycle.window, lifecycle.AwaitWindow()
	}
	if lifecycle.window == "" {
		return "", fmt.Errorf("ready window is not known")
	}
	data, err := lifecycle.client(lifecycle.window).Call("window.open", map[string]any{
		"root": lifecycle.config.Workspace, "focus": lifecycle.config.Focus,
	})
	if err != nil {
		return "", err
	}
	window, _ := data["label"].(string)
	if window == "" {
		return "", fmt.Errorf("window.open returned no label")
	}
	lifecycle.window = window
	if _, err := lifecycle.client(window).Call("window_renderer_wait", map[string]any{"targetWindow": window, "timeoutMs": 45000}); err != nil {
		return "", err
	}
	return window, lifecycle.AwaitWindow()
}

func (lifecycle *Lifecycle) AwaitWindow() error {
	if lifecycle.window == "" {
		return fmt.Errorf("workspace window is not known")
	}
	_, err := lifecycle.client(lifecycle.window).Call("app.boot.wait", map[string]any{"timeoutMs": 45000})
	if err == nil {
		_, err = lifecycle.client(lifecycle.window).Call("plugin.boot.wait", map[string]any{"timeoutMs": 45000})
	}
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w; recent activity: %s", err, lifecycle.recentActivity())
}

func (lifecycle *Lifecycle) recentActivity() string {
	value, err := lifecycle.client(lifecycle.window).CallValue("activity_recent", map[string]any{"limit": 20})
	if err != nil {
		return err.Error()
	}
	body, err := json.Marshal(value)
	if err != nil {
		return err.Error()
	}
	return string(body)
}

func (lifecycle *Lifecycle) Shutdown() error {
	if lifecycle.cmd == nil {
		return nil
	}
	window := lifecycle.window
	if window == "" {
		window = "main"
	}
	_, commandErr := lifecycle.client(window).Call("app.shutdown.commit", map[string]any{})
	if lifecycle.stdin != nil {
		_ = lifecycle.stdin.Close()
		lifecycle.stdin = nil
	}
	done := make(chan error, 1)
	go func() { done <- lifecycle.cmd.Wait() }()
	select {
	case waitErr := <-done:
		lifecycle.cmd = nil
		_ = lifecycle.log.Close()
		lifecycle.log = nil
		if commandErr != nil {
			return commandErr
		}
		return waitErr
	case <-time.After(20 * time.Second):
		_ = lifecycle.cmd.Process.Kill()
		<-done
		lifecycle.cmd = nil
		_ = lifecycle.log.Close()
		lifecycle.log = nil
		return fmt.Errorf("application did not stop within 20 seconds: %v: %s", commandErr, lifecycle.lastLog())
	}
}

func (lifecycle *Lifecycle) Finish() error {
	pid := 0
	if lifecycle.cmd != nil && lifecycle.cmd.Process != nil {
		pid = lifecycle.cmd.Process.Pid
	}
	before, err := lifecycle.ownershipSnapshot()
	if err != nil {
		return err
	}
	if err := lifecycle.stopTestSidecars(); err != nil {
		return err
	}
	afterSidecarStop, err := lifecycle.ownershipSnapshot()
	if err != nil {
		return err
	}
	shutdownErr := lifecycle.Shutdown()
	report := map[string]any{
		"process": map[string]any{
			"pid": pid, "identifier": lifecycle.config.Identifier,
			"socket": lifecycle.config.Socket, "home": lifecycle.config.Home,
			"runtime": lifecycle.config.Runtime,
		},
		"beforeShutdown":    before,
		"afterSidecarStop":  afterSidecarStop,
		"applicationExited": lifecycle.cmd == nil,
		"gracefulShutdown":  shutdownErr == nil,
	}
	if shutdownErr != nil {
		report["shutdownError"] = shutdownErr.Error()
	}
	body, marshalErr := json.MarshalIndent(report, "", "  ")
	if marshalErr == nil {
		marshalErr = os.WriteFile(
			filepath.Join(lifecycle.config.EvidenceDir, "process-window-ownership.json"),
			append(body, '\n'), 0o600,
		)
	}
	if marshalErr != nil {
		return marshalErr
	}
	return shutdownErr
}

func (lifecycle *Lifecycle) ownershipSnapshot() (map[string]any, error) {
	cli := lifecycle.Client()
	environment, err := cli.Call("app.environment", map[string]any{})
	if err != nil {
		return nil, err
	}
	windows, err := cli.CallValue("window_list", map[string]any{})
	if err != nil {
		return nil, err
	}
	monitors, err := cli.Call("window.monitors", map[string]any{})
	if err != nil {
		return nil, err
	}
	input, err := cli.Call("window.input.state", map[string]any{})
	if err != nil {
		return nil, err
	}
	sidecars, err := cli.Call("sidecar_status", map[string]any{})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"environment": environment,
		"windows":     windows,
		"monitors":    monitors,
		"input":       input,
		"sidecars":    sidecars,
	}, nil
}

func (lifecycle *Lifecycle) Close() {
	if lifecycle.cmd == nil {
		return
	}
	_ = lifecycle.stopTestSidecars()
	_ = lifecycle.cmd.Process.Kill()
	_, _ = lifecycle.cmd.Process.Wait()
	lifecycle.cmd = nil
	if lifecycle.stdin != nil {
		_ = lifecycle.stdin.Close()
	}
	if lifecycle.log != nil {
		_ = lifecycle.log.Close()
	}
}

func (lifecycle *Lifecycle) stopTestSidecars() error {
	return quiesceTestRuntime(lifecycle.client(lifecycle.window))
}

func quiesceTestRuntime(cli commandCaller) error {
	if err := closeTestPluginTabs(cli); err != nil {
		return err
	}
	plugins, err := cli.Call("plugin.list", map[string]any{})
	if err != nil {
		return err
	}
	for _, id := range enabledPluginNames(plugins) {
		if _, err := cli.Call("plugin.disable", map[string]any{"id": id}); err != nil {
			return fmt.Errorf("disable plugin %s: %w", id, err)
		}
	}
	value, err := cli.Call("sidecar_status", map[string]any{})
	if err != nil {
		return err
	}
	pids := ownedSidecarPIDs(value)
	for _, name := range ownedSidecarNames(value) {
		stopped, err := cli.Call("sidecar_stop", map[string]any{"name": name})
		if err != nil {
			return fmt.Errorf("stop sidecar %s: %w", name, err)
		}
		running, ok := stopped["running"].(bool)
		if !ok {
			return fmt.Errorf("stop sidecar %s returned no running status", name)
		}
		if running {
			return fmt.Errorf("sidecar %s still reports running after stop", name)
		}
	}
	after, err := cli.Call("sidecar_status", map[string]any{})
	if err != nil {
		return err
	}
	if remaining := ownedSidecarNames(after); len(remaining) > 0 {
		return fmt.Errorf("sidecar ownership remains after stop: %v", remaining)
	}
	for _, pid := range pids {
		gone, err := processGone(pid)
		if err != nil {
			return fmt.Errorf("read sidecar process %d after stop: %w", pid, err)
		}
		if !gone {
			return fmt.Errorf("sidecar process %d still exists after stop", pid)
		}
	}
	return nil
}

func closeTestPluginTabs(cli commandCaller) error {
	tree, err := cli.Call("state.tree", map[string]any{})
	if err != nil {
		return err
	}
	unique := map[string]struct{}{}
	workspaces, _ := tree["workspaces"].([]any)
	for _, workspaceValue := range workspaces {
		workspace, _ := workspaceValue.(map[string]any)
		spaces, _ := workspace["spaces"].([]any)
		for _, spaceValue := range spaces {
			space, _ := spaceValue.(map[string]any)
			panes, _ := space["panes"].([]any)
			for _, paneValue := range panes {
				pane, _ := paneValue.(map[string]any)
				tabs, _ := pane["tabs"].([]any)
				for _, tabValue := range tabs {
					tab, _ := tabValue.(map[string]any)
					id, _ := tab["id"].(string)
					plugin, _ := tab["plugin"].(string)
					if id != "" && plugin != "" {
						unique[id] = struct{}{}
					}
				}
			}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, err := cli.Call("tab.close", map[string]any{"tab": id}); err != nil {
			return fmt.Errorf("close plugin tab %s: %w", id, err)
		}
	}
	return nil
}

func enabledPluginNames(status map[string]any) []string {
	entries, _ := status["plugins"].([]any)
	names := []string{}
	for _, item := range entries {
		plugin, _ := item.(map[string]any)
		id, _ := plugin["id"].(string)
		if id != "" && plugin["status"] == "enabled" {
			names = append(names, id)
		}
	}
	sort.Strings(names)
	return names
}

func ownedSidecarNames(status map[string]any) []string {
	unique := map[string]struct{}{}
	for _, field := range []string{"open", "recorded"} {
		entries, _ := status[field].([]any)
		for _, item := range entries {
			entry, _ := item.(map[string]any)
			name, _ := entry["name"].(string)
			if name == "" {
				name, _ = entry["Name"].(string)
			}
			if name != "" {
				unique[name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ownedSidecarPIDs(status map[string]any) []uint32 {
	unique := map[uint32]struct{}{}
	for _, field := range []string{"open", "recorded"} {
		entries, _ := status[field].([]any)
		for _, item := range entries {
			entry, _ := item.(map[string]any)
			pid, ok := exactInt(entry["pid"])
			if ok && pid > 0 && uint64(pid) <= uint64(^uint32(0)) {
				unique[uint32(pid)] = struct{}{}
			}
		}
	}
	pids := make([]uint32, 0, len(unique))
	for pid := range unique {
		pids = append(pids, pid)
	}
	sort.Slice(pids, func(left, right int) bool { return pids[left] < pids[right] })
	return pids
}

func (lifecycle *Lifecycle) Client() CLI {
	return lifecycle.client(lifecycle.window)
}

func (lifecycle *Lifecycle) client(window string) CLI {
	return CLI{Path: lifecycle.config.CLI, Socket: lifecycle.config.Socket, Window: window, EvidenceDir: lifecycle.config.EvidenceDir}
}

func (lifecycle *Lifecycle) lastLog() string {
	body, err := os.ReadFile(filepath.Join(lifecycle.config.EvidenceDir, "application.log"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return strings.Join(lines, "\n")
}

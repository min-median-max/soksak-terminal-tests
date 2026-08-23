package system

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type LifecycleConfig struct {
	App         string
	CLI         string
	Socket      string
	Home        string
	Runtime     string
	Workspace   string
	EvidenceDir string
	Identifier  string
}

type Lifecycle struct {
	config LifecycleConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	log    *os.File
	window string
}

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
		"SOKSAK_UNATTENDED=1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = log.Close()
		return err
	}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		_ = log.Close()
		return err
	}
	lifecycle.cmd, lifecycle.stdin, lifecycle.log = cmd, stdin, log
	expectedWindow := lifecycle.window
	deadline := time.Now().Add(45 * time.Second)
	var readinessErr error
	for time.Now().Before(deadline) {
		value, err := lifecycle.client("main").CallValue("window_list", map[string]any{})
		if err == nil {
			windows, _ := value.([]any)
			for _, candidate := range windows {
				window, _ := candidate.(string)
				if window == "" || (expectedWindow != "" && window != expectedWindow) {
					continue
				}
				if _, readinessErr = lifecycle.client(window).Call("window_renderer_wait", map[string]any{"targetWindow": window, "timeoutMs": 45000}); readinessErr != nil {
					continue
				}
				_, readinessErr = lifecycle.client(window).Call("app.boot.wait", map[string]any{"timeoutMs": 45000})
				if readinessErr == nil {
					lifecycle.window = window
					return nil
				}
			}
		} else {
			readinessErr = err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("application did not answer within 45 seconds: %v: %s", readinessErr, lifecycle.lastLog())
}

func (lifecycle *Lifecycle) OpenWorkspace() (string, error) {
	if lifecycle.window == "" {
		return "", fmt.Errorf("ready window is not known")
	}
	data, err := lifecycle.client(lifecycle.window).Call("window.open", map[string]any{
		"root": lifecycle.config.Workspace, "focus": false,
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
	value, err := lifecycle.client("main").CallValue("activity_recent", map[string]any{"limit": 20})
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
	lifecycle.stopTestSidecars()
	return lifecycle.Shutdown()
}

func (lifecycle *Lifecycle) Close() {
	if lifecycle.cmd == nil {
		return
	}
	lifecycle.stopTestSidecars()
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

func (lifecycle *Lifecycle) stopTestSidecars() {
	value, err := lifecycle.client("main").Call("sidecar_status", map[string]any{})
	if err != nil {
		return
	}
	started, _ := value["open"].([]any)
	for _, item := range started {
		open, _ := item.(map[string]any)
		name, _ := open["Name"].(string)
		if name == "" {
			name, _ = open["name"].(string)
		}
		if name != "" {
			_, _ = lifecycle.client("main").Call("sidecar_stop", map[string]any{"name": name})
		}
	}
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

package system

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

type CLI struct {
	Path        string
	Socket      string
	Window      string
	EvidenceDir string
}

type response struct {
	Code string         `json:"code"`
	Data map[string]any `json:"data"`
}

func (cli CLI) Call(command string, params map[string]any) (map[string]any, error) {
	if !filepath.IsAbs(cli.Path) {
		return nil, fmt.Errorf("CLI path must be absolute: %s", cli.Path)
	}
	if cli.Window == "" {
		return nil, fmt.Errorf("window is required")
	}
	args := []string{}
	if cli.Socket != "" {
		if !filepath.IsAbs(cli.Socket) {
			return nil, fmt.Errorf("socket path must be absolute: %s", cli.Socket)
		}
		args = append(args, "--socket", cli.Socket)
	}
	params = cloneMap(params)
	params["window"] = cli.Window
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	args = append(args, command, string(body))
	output, runErr := exec.Command(cli.Path, args...).CombinedOutput()
	return decodeCLIResponse(command, output, runErr)
}

func decodeCLIResponse(command string, output []byte, runErr error) (map[string]any, error) {
	if runErr != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", command, runErr, output)
	}
	var answer response
	if err := json.Unmarshal(output, &answer); err != nil {
		return nil, fmt.Errorf("%s returned invalid JSON: %w: %s", command, err, output)
	}
	if answer.Code != "OK" || answer.Data == nil {
		return nil, fmt.Errorf("%s returned an invalid success envelope: %s", command, output)
	}
	return answer.Data, nil
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source)+1)
	for key, value := range source {
		result[key] = value
	}
	return result
}

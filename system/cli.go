package system

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

type CLI struct {
	Path   string
	Socket string
	Window string
}

type response struct {
	OK      bool           `json:"ok"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
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
	var answer response
	if err := json.Unmarshal(output, &answer); err != nil {
		return nil, fmt.Errorf("%s returned invalid JSON: %w: %s", command, err, output)
	}
	if runErr != nil || !answer.OK {
		return nil, fmt.Errorf("%s failed (%s): %s", command, answer.Code, answer.Message)
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

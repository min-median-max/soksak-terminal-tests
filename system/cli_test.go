package system

import (
	"errors"
	"testing"
)

func TestCLIRequiresDeclaredAbsolutePathsAndWindow(t *testing.T) {
	for _, cli := range []CLI{
		{Path: "sok", Window: "w-1"},
		{Path: "/bin/sok"},
		{Path: "/bin/sok", Socket: "relative.sock", Window: "w-1"},
	} {
		if _, err := cli.Call("state.tree", nil); err == nil {
			t.Fatalf("accepted CLI %+v", cli)
		}
	}
}

func TestDecodeCLIResponseUsesThePublicSokOutput(t *testing.T) {
	value, err := decodeCLIResponse("settings_get", []byte(`{"code":"OK","data":{"revision":7}}`), nil)
	data, _ := value.(map[string]any)
	if err != nil || data["revision"] != float64(7) {
		t.Fatalf("data=%+v err=%v", data, err)
	}
	if _, err := decodeCLIResponse("settings_get", []byte("failure"), errors.New("exit status 1")); err == nil {
		t.Fatal("failed CLI process was accepted")
	}
	value, err = decodeCLIResponse("window_list", []byte(`{"code":"OK","data":["main"]}`), nil)
	windows, _ := value.([]any)
	if err != nil || len(windows) != 1 || windows[0] != "main" {
		t.Fatalf("windows=%+v err=%v", windows, err)
	}
}

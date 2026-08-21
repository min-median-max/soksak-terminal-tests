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
	data, err := decodeCLIResponse("composition_status", []byte(`{"code":"OK","data":{"plugins":7}}`), nil)
	if err != nil || data["plugins"] != float64(7) {
		t.Fatalf("data=%+v err=%v", data, err)
	}
	if _, err := decodeCLIResponse("composition_status", []byte("failure"), errors.New("exit status 1")); err == nil {
		t.Fatal("failed CLI process was accepted")
	}
}

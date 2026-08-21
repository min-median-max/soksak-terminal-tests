package system

import "testing"

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

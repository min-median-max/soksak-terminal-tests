package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestNativeKeyboardMatrixCapturesEveryVerifiedProvider(t *testing.T) {
	body, err := os.ReadFile("system/terminal_native_input_system_test.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	required := `captureTerminal(matrix.cli, "native-keyboard-"+plugin.ID)`
	if !strings.Contains(source, required) {
		t.Fatalf("native keyboard matrix does not call %s", required)
	}
}

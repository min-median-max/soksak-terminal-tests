package system

import (
	"encoding/base64"
	"fmt"
	"unicode/utf16"
)

func terminalPrintCommand(platform, marker string) (string, error) {
	switch platform {
	case "windows":
		return "echo " + marker, nil
	case "darwin", "linux":
		return "printf '%s\n' " + marker, nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func terminalHighOutputCommand(platform, marker string) (string, error) {
	switch platform {
	case "windows":
		script := "[Console]::Out.Write(('X' * 262144)); [Console]::Out.WriteLine(); [Console]::Out.WriteLine('" + marker + "')"
		return "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " + encodePowerShell(script), nil
	case "darwin", "linux":
		return "yes X | head -c 262144; printf '\\n%s\\n' " + marker, nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func encodePowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, len(encoded)*2)
	for index, value := range encoded {
		bytes[index*2] = byte(value)
		bytes[index*2+1] = byte(value >> 8)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

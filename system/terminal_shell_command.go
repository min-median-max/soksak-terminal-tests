package system

import (
	"encoding/base64"
	"fmt"
	"strings"
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
	prefix, suffix, err := splitMarker(marker)
	if err != nil {
		return "", err
	}
	switch platform {
	case "windows":
		script := "$prefix='" + prefix + "'; [Console]::Out.Write(('X' * 262144)); [Console]::Out.WriteLine(); [Console]::Out.WriteLine($prefix + '" + suffix + "')"
		return "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " + encodePowerShell(script), nil
	case "darwin", "linux":
		return "prefix=" + prefix + "; head -c 262144 /dev/zero | tr '\\0' X; printf '\\n%s%s\\n' \"$prefix\" " + suffix, nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func splitMarker(marker string) (string, string, error) {
	cut := strings.LastIndex(marker, "_") + 1
	if cut < 1 || cut >= len(marker) {
		return "", "", fmt.Errorf("marker has no suffix: %s", marker)
	}
	return marker[:cut], marker[cut:], nil
}

func detachedMarkerCommand(platform, marker, scheduled string) (string, error) {
	switch platform {
	case "windows":
		script := "Start-Sleep -Seconds 10; [Console]::Out.WriteLine(); [Console]::Out.WriteLine('" + marker + "')"
		return `start "" /b powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand ` +
			encodePowerShell(script) + " & echo " + scheduled + "\r", nil
	case "darwin", "linux":
		cut := strings.LastIndex(marker, "_") + 1
		if cut < 1 || cut >= len(marker) {
			return "", fmt.Errorf("detached marker has no suffix: %s", marker)
		}
		return "prefix=" + marker[:cut] + "; (sleep 10; printf '\n%s\n' \"${prefix}" + marker[cut:] +
			"\") & printf '%s\n' " + scheduled + "\r", nil
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

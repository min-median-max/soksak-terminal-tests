package system

import (
	"encoding/base64"
	"fmt"
	"unicode/utf16"
)

func terminalPrintCommand(platform, marker string) (string, error) {
	payload := base64.StdEncoding.EncodeToString([]byte(marker + "\n"))
	switch platform {
	case "windows":
		return "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " +
			encodePowerShell("[Console]::Out.WriteLine('"+marker+"')"), nil
	case "darwin", "linux":
		return "printf %s " + payload + " | base64 -d", nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func terminalHighOutputCommand(platform, marker string) (string, error) {
	payload := base64.StdEncoding.EncodeToString([]byte(marker + "\n"))
	switch platform {
	case "windows":
		script := fmt.Sprintf("[Console]::Out.Write(('X' * %d)); [Console]::Out.WriteLine(); [Console]::Out.WriteLine('%s')", highOutputPayloadBytes, marker)
		return "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " + encodePowerShell(script), nil
	case "darwin", "linux":
		return fmt.Sprintf("head -c %d /dev/zero | tr '\\0' X; printf '\\n'; printf %%s %s | base64 -d", highOutputPayloadBytes, payload), nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func terminalPaletteCommand(platform, marker string) (string, error) {
	payload := base64.StdEncoding.EncodeToString([]byte(marker + "\n"))
	switch platform {
	case "windows":
		script := "[Console]::Out.Write('`e[41m R `e[42m G `e[44m B `e[101m r `e[102m g `e[104m b `e[0m'); [Console]::Out.WriteLine('" + marker + "')"
		return "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " + encodePowerShell(script), nil
	case "darwin", "linux":
		return "printf '\\033[41m R \\033[42m G \\033[44m B \\033[101m r \\033[102m g \\033[104m b \\033[0m'; printf %s " + payload + " | base64 -d", nil
	default:
		return "", fmt.Errorf("unsupported terminal platform: %s", platform)
	}
}

func detachedMarkerCommand(platform, marker, scheduled string) (string, error) {
	switch platform {
	case "windows":
		script := "Start-Sleep -Seconds 10; [Console]::Out.WriteLine(); [Console]::Out.WriteLine('" + marker + "')"
		ready := "[Console]::Out.WriteLine('" + scheduled + "')"
		return `start "" /b powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand ` +
			encodePowerShell(script) + " & powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand " + encodePowerShell(ready), nil
	case "darwin", "linux":
		markerPayload := base64.StdEncoding.EncodeToString([]byte(marker + "\n"))
		readyPayload := base64.StdEncoding.EncodeToString([]byte(scheduled + "\n"))
		return "(sleep 10; printf '\\n'; printf %s " + markerPayload + " | base64 -d) & printf %s " + readyPayload + " | base64 -d", nil
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

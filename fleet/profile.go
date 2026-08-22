package fleet

import "fmt"

type Plugin struct {
	ID          string
	Requirement string
	Sidecar     string
}

type Profile struct {
	Platform         string
	Target           string
	Plugins          []Plugin
	RecoverySidecars []string
}

var fullPlugins = []Plugin{
	{"soksak-plugin-terminal-alacritty", "terminal-alacritty", "soksak-sidecar-terminal-alacritty"},
	{"soksak-plugin-terminal-ghostty", "terminal-ghostty", "soksak-sidecar-terminal-ghostty"},
	{"soksak-plugin-terminal-kitty", "terminal-kitty", "soksak-sidecar-terminal-kitty"},
	{"soksak-plugin-terminal-shitty", "terminal-shitty", "soksak-sidecar-terminal-shitty"},
	{"soksak-plugin-terminal-vt100", "terminal-vt100", "soksak-sidecar-terminal-vt100"},
	{"soksak-plugin-terminal-wezterm", "terminal-wezterm", "soksak-sidecar-terminal-wezterm"},
	{"soksak-plugin-terminal-xterm", "terminal-vt100", "soksak-sidecar-terminal-vt100"},
}

var fullRecoverySidecars = []string{
	"soksak-sidecar-terminal-alacritty", "soksak-sidecar-terminal-ghostty",
	"soksak-sidecar-terminal-kitty", "soksak-sidecar-terminal-shitty",
	"soksak-sidecar-terminal-vt100", "soksak-sidecar-terminal-wezterm",
}

func ForPlatform(platform string) (Profile, error) {
	switch platform {
	case "darwin":
		return Profile{Platform: platform, Target: "aarch64-apple-darwin", Plugins: clonePlugins(fullPlugins), RecoverySidecars: cloneStrings(fullRecoverySidecars)}, nil
	case "linux":
		return Profile{Platform: platform, Target: "aarch64-unknown-linux-gnu", Plugins: clonePlugins(fullPlugins), RecoverySidecars: cloneStrings(fullRecoverySidecars)}, nil
	case "windows":
		return Profile{
			Platform: platform, Target: "x86_64-pc-windows-msvc",
			Plugins:          []Plugin{fullPlugins[0], fullPlugins[1], fullPlugins[4], fullPlugins[5], fullPlugins[6]},
			RecoverySidecars: []string{fullRecoverySidecars[0], fullRecoverySidecars[1], fullRecoverySidecars[4], fullRecoverySidecars[5]},
		}, nil
	default:
		return Profile{}, fmt.Errorf("unsupported terminal fleet platform: %s", platform)
	}
}

func clonePlugins(values []Plugin) []Plugin { return append([]Plugin(nil), values...) }
func cloneStrings(values []string) []string { return append([]string(nil), values...) }

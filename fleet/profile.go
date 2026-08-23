package fleet

import "fmt"

type Plugin struct {
	ID      string
	Sidecar string
}

type Profile struct {
	Platform         string
	Target           string
	Plugins          []Plugin
	RecoverySidecars []string
}

var fullPlugins = []Plugin{
	{"soksak-plugin-terminal-alacritty", "soksak-sidecar-terminal-alacritty"},
	{"soksak-plugin-terminal-ghostty", "soksak-sidecar-terminal-ghostty"},
	{"soksak-plugin-terminal-kitty", "soksak-sidecar-terminal-kitty"},
	{"soksak-plugin-terminal-shitty", "soksak-sidecar-terminal-shitty"},
	{"soksak-plugin-terminal-vt100", "soksak-sidecar-terminal-vt100"},
	{"soksak-plugin-terminal-wezterm", "soksak-sidecar-terminal-wezterm"},
	{"soksak-plugin-terminal-xterm", "soksak-sidecar-terminal-vt100"},
}

var fullRecoverySidecars = []string{
	"soksak-sidecar-terminal-alacritty", "soksak-sidecar-terminal-ghostty",
	"soksak-sidecar-terminal-kitty", "soksak-sidecar-terminal-shitty",
	"soksak-sidecar-terminal-vt100", "soksak-sidecar-terminal-wezterm",
}

func ForTarget(platform, target string) (Profile, error) {
	switch {
	case platform == "darwin" && (target == "aarch64-apple-darwin" || target == "x86_64-apple-darwin"):
		return Profile{Platform: platform, Target: target, Plugins: clonePlugins(fullPlugins), RecoverySidecars: cloneStrings(fullRecoverySidecars)}, nil
	case platform == "linux" && (target == "aarch64-unknown-linux-gnu" || target == "x86_64-unknown-linux-gnu"):
		return Profile{Platform: platform, Target: target, Plugins: clonePlugins(fullPlugins), RecoverySidecars: cloneStrings(fullRecoverySidecars)}, nil
	case platform == "windows" && target == "x86_64-pc-windows-msvc":
		return Profile{
			Platform: platform, Target: target,
			Plugins:          []Plugin{fullPlugins[0], fullPlugins[1], fullPlugins[4], fullPlugins[5], fullPlugins[6]},
			RecoverySidecars: []string{fullRecoverySidecars[0], fullRecoverySidecars[1], fullRecoverySidecars[4], fullRecoverySidecars[5]},
		}, nil
	default:
		return Profile{}, fmt.Errorf("unsupported terminal fleet target: %s/%s", platform, target)
	}
}

func clonePlugins(values []Plugin) []Plugin { return append([]Plugin(nil), values...) }
func cloneStrings(values []string) []string { return append([]string(nil), values...) }

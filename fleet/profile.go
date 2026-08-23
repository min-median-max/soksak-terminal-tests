package fleet

import "fmt"

type Component struct{ ID, Version string }
type Plugin struct {
	Component
	Sidecar string
}

type Profile struct {
	Platform string
	Target   string
	Plugins  []Plugin
	Sidecars []Component
}

var fullPlugins = []Plugin{
	{Component{"soksak-plugin-terminal-alacritty", "0.0.11"}, "soksak-sidecar-terminal-alacritty"},
	{Component{"soksak-plugin-terminal-ghostty", "0.0.12"}, "soksak-sidecar-terminal-ghostty"},
	{Component{"soksak-plugin-terminal-kitty", "0.0.11"}, "soksak-sidecar-terminal-kitty"},
	{Component{"soksak-plugin-terminal-shitty", "0.0.11"}, "soksak-sidecar-terminal-shitty"},
	{Component{"soksak-plugin-terminal-vt100", "0.0.11"}, "soksak-sidecar-terminal-vt100"},
	{Component{"soksak-plugin-terminal-wezterm", "0.0.11"}, "soksak-sidecar-terminal-wezterm"},
	{Component{"soksak-plugin-terminal-xterm", "0.0.18"}, "soksak-sidecar-terminal-vt100"},
}

var fullSidecars = []Component{
	{"soksak-sidecar-pty", "0.0.6"},
	{"soksak-sidecar-terminal-alacritty", "0.0.12"}, {"soksak-sidecar-terminal-ghostty", "0.0.12"},
	{"soksak-sidecar-terminal-kitty", "0.0.7"}, {"soksak-sidecar-terminal-shitty", "0.0.7"},
	{"soksak-sidecar-terminal-vt100", "0.0.11"}, {"soksak-sidecar-terminal-wezterm", "0.0.11"},
}

func ForTarget(platform, target string) (Profile, error) {
	switch {
	case platform == "darwin" && (target == "aarch64-apple-darwin" || target == "x86_64-apple-darwin"):
		return Profile{Platform: platform, Target: target, Plugins: clonePlugins(fullPlugins), Sidecars: cloneComponents(fullSidecars)}, nil
	case platform == "linux" && (target == "aarch64-unknown-linux-gnu" || target == "x86_64-unknown-linux-gnu"):
		return Profile{Platform: platform, Target: target, Plugins: clonePlugins(fullPlugins), Sidecars: cloneComponents(fullSidecars)}, nil
	case platform == "windows" && target == "x86_64-pc-windows-msvc":
		return Profile{
			Platform: platform, Target: target,
			Plugins:  []Plugin{fullPlugins[0], fullPlugins[1], fullPlugins[4], fullPlugins[5], fullPlugins[6]},
			Sidecars: []Component{fullSidecars[0], fullSidecars[1], fullSidecars[2], fullSidecars[5], fullSidecars[6]},
		}, nil
	default:
		return Profile{}, fmt.Errorf("unsupported terminal fleet target: %s/%s", platform, target)
	}
}

func clonePlugins(values []Plugin) []Plugin          { return append([]Plugin(nil), values...) }
func cloneComponents(values []Component) []Component { return append([]Component(nil), values...) }
func (profile Profile) RecoverySidecarIDs() []string {
	ids := []string{}
	for _, component := range profile.Sidecars {
		if component.ID != "soksak-sidecar-pty" {
			ids = append(ids, component.ID)
		}
	}
	return ids
}

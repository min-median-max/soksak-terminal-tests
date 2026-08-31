package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type Component struct {
	ID, Version, ReleaseSHA256 string
	ReleaseSize                int64
}
type Plugin struct {
	Component
	Sidecar string
	Program string
}

type Profile struct {
	Platform string
	Target   string
	Registry Component
	Plugins  []Plugin
	Sidecars []Component
}

var officialRegistry = Component{
	ID: "registry-14", Version: "14", ReleaseSize: 7915,
	ReleaseSHA256: "a8f085d09af701a463af0345983d6b58ab04881e3a8b319161211fc5098d82b9",
}

var fullPlugins = []Plugin{
	{Component: Component{ID: "soksak-plugin-terminal-alacritty", Version: "0.0.15", ReleaseSize: 2278, ReleaseSHA256: "29b5adacba5fb3132e0b8805dcf43161dab81896a3b362b68d47e6c331147020"}, Sidecar: "soksak-sidecar-terminal-alacritty", Program: "terminal-alacritty"},
	{Component: Component{ID: "soksak-plugin-terminal-ghostty", Version: "0.0.16", ReleaseSize: 2258, ReleaseSHA256: "2eb4648b81d6a404471e66c91881d81c8d156121fb4cebd7fb260ce2fe75968b"}, Sidecar: "soksak-sidecar-terminal-ghostty", Program: "terminal-ghostty"},
	{Component: Component{ID: "soksak-plugin-terminal-kitty", Version: "0.0.15", ReleaseSize: 2236, ReleaseSHA256: "a69390b94904b6aea878e9007d4299d63b3bd93d3a9deccea6163f7a3077200d"}, Sidecar: "soksak-sidecar-terminal-kitty", Program: "terminal-kitty"},
	{Component: Component{ID: "soksak-plugin-terminal-shitty", Version: "0.0.15", ReleaseSize: 2246, ReleaseSHA256: "9cf7e0a6a634685006001bbde118579f78e0e012f1bece0b6ec53943d78faa15"}, Sidecar: "soksak-sidecar-terminal-shitty", Program: "terminal-shitty"},
	{Component: Component{ID: "soksak-plugin-terminal-vt100", Version: "0.0.15", ReleaseSize: 2238, ReleaseSHA256: "43737c8b0fa4088c5e71462bde92e4b57103e8be55977df083f02daee1ad74f9"}, Sidecar: "soksak-sidecar-terminal-vt100", Program: "terminal-vt100"},
	{Component: Component{ID: "soksak-plugin-terminal-wezterm", Version: "0.0.15", ReleaseSize: 2258, ReleaseSHA256: "11d24bd906babe20305d539347616a76b9c38a192df0592f2b50dd80fab7bfed"}, Sidecar: "soksak-sidecar-terminal-wezterm", Program: "terminal-wezterm"},
	{Component: Component{ID: "soksak-plugin-terminal-xterm", Version: "0.0.22", ReleaseSize: 2239, ReleaseSHA256: "886ef173a7919e8f7bfcb449db12e8b4b73c1c0cfc00005f1a7b5a3b0bd771fe"}, Sidecar: "soksak-sidecar-terminal-vt100", Program: "terminal-xterm"},
}

var fullSidecars = []Component{
	{ID: "soksak-sidecar-pty", Version: "0.0.7", ReleaseSize: 2996, ReleaseSHA256: "58e102c1943c4d478a3f5d1d39d19c0dba95b521de76a1fb9205469cd1650c37"},
	{ID: "soksak-sidecar-terminal-alacritty", Version: "0.0.13", ReleaseSize: 3246, ReleaseSHA256: "01838a47dcde2058382ac948bdc7523963ac5aaa7c0ad11574d56e8c48fa1f6f"},
	{ID: "soksak-sidecar-terminal-ghostty", Version: "0.0.13", ReleaseSize: 3216, ReleaseSHA256: "6b1f93e333b79037e949dd4a367247d207d2068c5670e00df6ab0635a365d2da"},
	{ID: "soksak-sidecar-terminal-kitty", Version: "0.0.8", ReleaseSize: 2788, ReleaseSHA256: "2594bb7ba6b1a4e5c087b495cfc0f0e89ea985bc026a1baae76bbdda318e2a3c"},
	{ID: "soksak-sidecar-terminal-shitty", Version: "0.0.8", ReleaseSize: 2794, ReleaseSHA256: "f612364def7b03afa678400a42127cb81939a1e5042e9525b9d471f093792139"},
	{ID: "soksak-sidecar-terminal-vt100", Version: "0.0.12", ReleaseSize: 3182, ReleaseSHA256: "8d42f43323e24ae1923e1874d48f8c9060a4cbc70a532fec51f7aab096b1f83d"},
	{ID: "soksak-sidecar-terminal-wezterm", Version: "0.0.12", ReleaseSize: 3219, ReleaseSHA256: "70ac8853e3fe5b422f473c649e516f263588eb7b5e3e97a30449b94b23eb3f39"},
}

func ForTarget(platform, target string) (Profile, error) {
	switch {
	case platform == "darwin" && (target == "aarch64-apple-darwin" || target == "x86_64-apple-darwin"):
		return Profile{Platform: platform, Target: target, Registry: officialRegistry, Plugins: clonePlugins(fullPlugins), Sidecars: cloneComponents(fullSidecars)}, nil
	case platform == "linux" && (target == "aarch64-unknown-linux-gnu" || target == "x86_64-unknown-linux-gnu"):
		return Profile{Platform: platform, Target: target, Registry: officialRegistry, Plugins: clonePlugins(fullPlugins), Sidecars: cloneComponents(fullSidecars)}, nil
	case platform == "windows" && target == "x86_64-pc-windows-msvc":
		return Profile{
			Platform: platform, Target: target, Registry: officialRegistry,
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

func (profile Profile) Fingerprint() (string, error) {
	lines := []string{"platform/" + profile.Platform + "/" + profile.Target}
	appendComponent := func(kind string, component Component) error {
		if component.ID == "" || component.Version == "" || component.ReleaseSize <= 0 ||
			len(component.ReleaseSHA256) != 64 {
			return fmt.Errorf("%s %s has no immutable release identity", kind, component.ID)
		}
		if _, err := hex.DecodeString(component.ReleaseSHA256); err != nil {
			return fmt.Errorf("%s %s release digest: %w", kind, component.ID, err)
		}
		lines = append(lines, fmt.Sprintf("%s/%s/%s/%d/%s", kind, component.ID, component.Version, component.ReleaseSize, component.ReleaseSHA256))
		return nil
	}
	if err := appendComponent("registry", profile.Registry); err != nil {
		return "", err
	}
	for _, plugin := range profile.Plugins {
		if err := appendComponent("plugin", plugin.Component); err != nil {
			return "", err
		}
	}
	for _, sidecar := range profile.Sidecars {
		if err := appendComponent("sidecar", sidecar); err != nil {
			return "", err
		}
	}
	sort.Strings(lines[1:])
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(digest[:]), nil
}

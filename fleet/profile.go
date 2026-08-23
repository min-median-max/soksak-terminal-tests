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
}

type Profile struct {
	Platform string
	Target   string
	Registry Component
	Plugins  []Plugin
	Sidecars []Component
}

var officialRegistry = Component{
	ID: "registry-13", Version: "13", ReleaseSize: 7915,
	ReleaseSHA256: "598121d12066dec61eb8347cd72b3590741ba0edebcc4e82ab021de908ca61fa",
}

var fullPlugins = []Plugin{
	{Component: Component{ID: "soksak-plugin-terminal-alacritty", Version: "0.0.14", ReleaseSize: 2278, ReleaseSHA256: "dd4c5f14ba91303d83e884a6220d68f7c57039aa91f99284c57bbbdbda5d38fa"}, Sidecar: "soksak-sidecar-terminal-alacritty"},
	{Component: Component{ID: "soksak-plugin-terminal-ghostty", Version: "0.0.15", ReleaseSize: 2258, ReleaseSHA256: "a6e2d88f83fc85ad3829c8e7fad875b25a9ac6802a4c87a9befb244a77ae5816"}, Sidecar: "soksak-sidecar-terminal-ghostty"},
	{Component: Component{ID: "soksak-plugin-terminal-kitty", Version: "0.0.14", ReleaseSize: 2236, ReleaseSHA256: "74f36c5b212fd57d3e499b73783b940b5c2a7ae5f6fb4db1586aed4bab9e8364"}, Sidecar: "soksak-sidecar-terminal-kitty"},
	{Component: Component{ID: "soksak-plugin-terminal-shitty", Version: "0.0.14", ReleaseSize: 2246, ReleaseSHA256: "5d3a3a00ca550404458e3f1a8daa4ead5c17778fd7fc56e4f187e92f02602537"}, Sidecar: "soksak-sidecar-terminal-shitty"},
	{Component: Component{ID: "soksak-plugin-terminal-vt100", Version: "0.0.14", ReleaseSize: 2238, ReleaseSHA256: "9057cd8f2726dac46260e15cbe38c93f5092c9d72866cad0216962f9dcb0d297"}, Sidecar: "soksak-sidecar-terminal-vt100"},
	{Component: Component{ID: "soksak-plugin-terminal-wezterm", Version: "0.0.14", ReleaseSize: 2258, ReleaseSHA256: "dae7421e32748c20ae70e615f189206699d469deab32a1de78fcf114c576f76d"}, Sidecar: "soksak-sidecar-terminal-wezterm"},
	{Component: Component{ID: "soksak-plugin-terminal-xterm", Version: "0.0.21", ReleaseSize: 2239, ReleaseSHA256: "0bfcad93004633fd4e5725e12f5d1d59d9f439b7d5a6f2713c987f69c130f477"}, Sidecar: "soksak-sidecar-terminal-vt100"},
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

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
	ID: "registry-12", Version: "12", ReleaseSize: 7915,
	ReleaseSHA256: "c19eec7208b0013df6aca6426b7d2b593d04957ace986f4e3a0a2ba8d4d740bf",
}

var fullPlugins = []Plugin{
	{Component: Component{ID: "soksak-plugin-terminal-alacritty", Version: "0.0.13", ReleaseSize: 2278, ReleaseSHA256: "7e4e0426c7711bac61252e58e821373e968aab91e00fcb57e14bae4cf593b5a3"}, Sidecar: "soksak-sidecar-terminal-alacritty"},
	{Component: Component{ID: "soksak-plugin-terminal-ghostty", Version: "0.0.14", ReleaseSize: 2258, ReleaseSHA256: "8eef8e7ef3e8d835bb7ffd52a096735b8932534c21df7732c1487e1d3b609994"}, Sidecar: "soksak-sidecar-terminal-ghostty"},
	{Component: Component{ID: "soksak-plugin-terminal-kitty", Version: "0.0.13", ReleaseSize: 2236, ReleaseSHA256: "72114f485d0805df741164d401eabdf799340ecb5f37d631ecdaba71722890c7"}, Sidecar: "soksak-sidecar-terminal-kitty"},
	{Component: Component{ID: "soksak-plugin-terminal-shitty", Version: "0.0.13", ReleaseSize: 2246, ReleaseSHA256: "1bc623b6f2e39a290be1ea74b11601816b0f64abadcb22e7a07243aa50413f15"}, Sidecar: "soksak-sidecar-terminal-shitty"},
	{Component: Component{ID: "soksak-plugin-terminal-vt100", Version: "0.0.13", ReleaseSize: 2238, ReleaseSHA256: "06065b7afbb609bf0dcd7f4a2fdb2d8780e69c071f60121649ac5207ba1b628a"}, Sidecar: "soksak-sidecar-terminal-vt100"},
	{Component: Component{ID: "soksak-plugin-terminal-wezterm", Version: "0.0.13", ReleaseSize: 2258, ReleaseSHA256: "9186cb81ca52d0bf407b793f7feb907cfdb17960887332af315c93eb741d5211"}, Sidecar: "soksak-sidecar-terminal-wezterm"},
	{Component: Component{ID: "soksak-plugin-terminal-xterm", Version: "0.0.20", ReleaseSize: 2239, ReleaseSHA256: "0e73249a45f64c0cf2bee9aa1181eefa5147a71073322f8063cab31f22d4b9cd"}, Sidecar: "soksak-sidecar-terminal-vt100"},
}

var fullSidecars = []Component{
	{ID: "soksak-sidecar-pty", Version: "0.0.6", ReleaseSize: 2996, ReleaseSHA256: "32e05911074b381222cbcac6004c76c93e73c9fe967df57b649af5abb5c69c40"},
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

package system

import "fmt"

func VerifyInstalledUI(cli CLI) error {
	plugins, err := cli.Call("plugin.list", map[string]any{})
	if err != nil {
		return err
	}
	rejected, _ := plugins["rejected"].([]any)
	if len(rejected) != 0 {
		return fmt.Errorf("plugin loader rejected %d plugins: %+v", len(rejected), rejected)
	}
	verified, err := cli.Call("ui.verify", map[string]any{})
	if err != nil {
		return err
	}
	failed, _ := verified["failed"].(float64)
	unanswered, _ := verified["unanswered"].(float64)
	if verified["passed"] != true || failed != 0 || unanswered != 0 {
		return fmt.Errorf("ui.verify failed: %+v", verified)
	}
	composition, err := cli.Call("surface.composition", map[string]any{})
	if err != nil {
		return err
	}
	for _, field := range []string{"displaced", "unapplied", "undeclared", "misparented"} {
		values, _ := composition[field].([]any)
		if len(values) != 0 {
			return fmt.Errorf("surface.composition %s=%+v", field, values)
		}
	}
	surfaces, _ := composition["surfaces"].([]any)
	for _, value := range surfaces {
		surface, _ := value.(map[string]any)
		covered, _ := surface["coveredFraction"].(float64)
		if covered != 0 {
			return fmt.Errorf("surface %v is %.2f%% covered", surface["id"], covered*100)
		}
	}
	alignment, err := cli.Call("layout.alignment", map[string]any{})
	if err != nil {
		return err
	}
	off, _ := alignment["worstOff"].(float64)
	if off > 2 {
		return fmt.Errorf("surface alignment is off by %.2fpx", off)
	}
	return nil
}

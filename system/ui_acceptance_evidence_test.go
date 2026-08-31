package system

import "testing"

func TestReadWorkspaceTitleEvidenceRequiresTextAndVisibleRect(t *testing.T) {
	snapshot := map[string]any{
		"count": float64(1),
		"nodes": []any{map[string]any{
			"selector": ".workspace-tab-title",
			"text":     "workspace 1",
			"rect": map[string]any{
				"x": float64(20), "y": float64(12), "w": float64(88), "h": float64(24),
			},
		}},
	}

	evidence, err := readWorkspaceTitleEvidence(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Selector != workspaceTitleSelector || evidence.Count != 1 {
		t.Fatalf("workspace title identity = %+v", evidence)
	}
	if len(evidence.Nodes) != 1 || evidence.Nodes[0].Text != "workspace 1" {
		t.Fatalf("workspace title text = %+v", evidence.Nodes)
	}
	if evidence.Nodes[0].X != 20 || evidence.Nodes[0].Y != 12 || evidence.Nodes[0].Width != 88 || evidence.Nodes[0].Height != 24 {
		t.Fatalf("workspace title rectangle = %+v", evidence.Nodes[0])
	}
}

func TestReadWorkspaceTitleEvidenceRejectsEmptyTextAndZeroSize(t *testing.T) {
	cases := map[string]struct {
		text  string
		width float64
	}{
		"empty-text": {text: "", width: 88},
		"zero-width": {text: "workspace 1", width: 0},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readWorkspaceTitleEvidence(map[string]any{
				"count": float64(1),
				"nodes": []any{map[string]any{
					"selector": workspaceTitleSelector,
					"text":     testCase.text,
					"rect": map[string]any{
						"x": float64(20), "y": float64(12), "w": testCase.width, "h": float64(24),
					},
				}},
			})
			if err == nil {
				t.Fatal("invalid workspace title evidence passed")
			}
		})
	}
}

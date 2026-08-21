package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ReportSpec = "soksak-spec-terminal-benchmark@0.0.1"

var Sidecars = []string{"alacritty", "ghostty", "kitty", "shitty", "vt100", "wezterm"}

type Report struct {
	Spec        string  `json:"spec"`
	Sidecar     string  `json:"sidecar"`
	FeedMBs     float64 `json:"feedMbS"`
	RehydrateMS float64 `json:"rehydrateMs"`
	PaintBytes  uint64  `json:"paintBytes"`
	ColdMS      float64 `json:"coldMs"`
	ColdBytes   uint64  `json:"coldBytes"`
	LiveBytes   uint64  `json:"liveBytes"`
	RSSBytes    uint64  `json:"rssBytes"`
	DemandMBs   float64 `json:"demandMbS"`
	GapBytes    uint64  `json:"gapBytes"`
	TailSeen    bool    `json:"tailSeen"`
}

func ReadReports(directory string) ([]Report, error) {
	if !filepath.IsAbs(directory) {
		return nil, fmt.Errorf("benchmark report directory must be absolute: %s", directory)
	}
	reports := make([]Report, 0, len(Sidecars))
	seen := map[string]bool{}
	for _, sidecar := range Sidecars {
		path := filepath.Join(directory, sidecar+".bench.json")
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		var report Report
		if err := decoder.Decode(&report); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return nil, fmt.Errorf("%s has trailing JSON", path)
		}
		expectedID := "soksak-sidecar-terminal-" + sidecar
		if report.Spec != ReportSpec || report.Sidecar != expectedID {
			return nil, fmt.Errorf("%s has identity %s/%s", path, report.Spec, report.Sidecar)
		}
		if seen[report.Sidecar] {
			return nil, fmt.Errorf("duplicate benchmark sidecar: %s", report.Sidecar)
		}
		seen[report.Sidecar] = true
		if report.FeedMBs <= 0 || report.DemandMBs <= 0 || report.RehydrateMS < 0 || report.ColdMS < 0 {
			return nil, fmt.Errorf("%s contains an invalid measurement", path)
		}
		if report.GapBytes != 0 || !report.TailSeen || report.FeedMBs < report.DemandMBs {
			return nil, fmt.Errorf("%s failed its absolute budget: feed=%.1f demand=%.1f gap=%d tail=%v",
				report.Sidecar, report.FeedMBs, report.DemandMBs, report.GapBytes, report.TailSeen)
		}
		reports = append(reports, report)
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].Sidecar < reports[j].Sidecar })
	return reports, nil
}

func Table(reports []Report) string {
	var output strings.Builder
	output.WriteString("provider feedMB/s demandMB/s gapBytes tail rehydrateMs coldMs paintKB rssMB\n")
	for _, report := range reports {
		fmt.Fprintf(&output, "%s %.1f %.1f %d %t %.2f %.2f %.1f %.1f\n",
			report.Sidecar, report.FeedMBs, report.DemandMBs, report.GapBytes, report.TailSeen,
			report.RehydrateMS, report.ColdMS, float64(report.PaintBytes)/1024, float64(report.RSSBytes)/1e6)
	}
	return output.String()
}

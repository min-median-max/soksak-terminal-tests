package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadsSixOwnerReportsAndBuildsAComparisonTable(t *testing.T) {
	directory := t.TempDir()
	for index, sidecar := range Sidecars {
		report := Report{
			Spec: ReportSpec, Sidecar: "soksak-sidecar-terminal-" + sidecar, FeedMBs: 100 + float64(index),
			RehydrateMS: 1, PaintBytes: 1024, ColdMS: 2, ColdBytes: 1024,
			LiveBytes: 2048, RSSBytes: 3_000_000,
		}
		body, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, sidecar+".bench.json"), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := ReadReports(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 6 {
		t.Fatalf("reports=%d", len(reports))
	}
	table := Table(reports)
	for _, sidecar := range Sidecars {
		if !strings.Contains(table, "soksak-sidecar-terminal-"+sidecar) {
			t.Fatalf("table has no %s: %s", sidecar, table)
		}
	}
}

func TestRejectsMissingProviderAndOwnerFloor(t *testing.T) {
	if _, err := ReadReports(t.TempDir()); err == nil {
		t.Fatal("missing reports were accepted")
	}
}

func TestRejectsTrailingReportData(t *testing.T) {
	directory := t.TempDir()
	for _, sidecar := range Sidecars {
		body := "{\"spec\":\"soksak-spec-terminal-benchmark@0.0.2\",\"sidecar\":\"soksak-sidecar-terminal-" + sidecar + "\",\"feedMbS\":100,\"rehydrateMs\":1,\"paintBytes\":1,\"coldMs\":1,\"coldBytes\":1,\"liveBytes\":1,\"rssBytes\":1}"
		if sidecar == Sidecars[0] {
			body += "{}"
		}
		if err := os.WriteFile(filepath.Join(directory, sidecar+".bench.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReadReports(directory); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

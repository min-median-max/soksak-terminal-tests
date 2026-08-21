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
	for index, provider := range Providers {
		report := Report{
			Spec: ReportSpec, Unit: provider, FeedMBs: 100 + float64(index), DemandMBs: 75,
			RehydrateMS: 1, PaintBytes: 1024, ColdMS: 2, ColdBytes: 1024,
			LiveBytes: 2048, RSSBytes: 3_000_000, GapBytes: 0, TailSeen: true,
		}
		body, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, provider+".bench.json"), body, 0o600); err != nil {
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
	for _, provider := range Providers {
		if !strings.Contains(table, provider) {
			t.Fatalf("table has no %s: %s", provider, table)
		}
	}
}

func TestRejectsMissingProviderAndMeasuredLoss(t *testing.T) {
	if _, err := ReadReports(t.TempDir()); err == nil {
		t.Fatal("missing reports were accepted")
	}
}

func TestRejectsTrailingReportData(t *testing.T) {
	directory := t.TempDir()
	for _, provider := range Providers {
		body := "{\"spec\":\"soksak-spec-terminal-benchmark@0.0.1\",\"unit\":\"" + provider + "\",\"feedMbS\":100,\"rehydrateMs\":1,\"paintBytes\":1,\"coldMs\":1,\"coldBytes\":1,\"liveBytes\":1,\"rssBytes\":1,\"demandMbS\":75,\"gapBytes\":0,\"tailSeen\":true}"
		if provider == Providers[0] {
			body += "{}"
		}
		if err := os.WriteFile(filepath.Join(directory, provider+".bench.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReadReports(directory); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

//go:build benchmark

package benchmark

import (
	"os"
	"testing"
)

func TestOwnerBenchmarkReports(t *testing.T) {
	directory := os.Getenv("SOKSAK_BENCH_REPORTS")
	if directory == "" {
		t.Fatal("SOKSAK_BENCH_REPORTS must name the owner-produced report directory")
	}
	reports, err := ReadReports(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + Table(reports))
}

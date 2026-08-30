package parallelism

import (
	"testing"
)

func TestConcurrencyVsParallelismBenchmark(t *testing.T) {
	res := RunComparisonBenchmark(8, 10000)

	if res.DurationSingleCore <= 0 || res.DurationMultiCore <= 0 {
		t.Errorf("Expected positive benchmark durations")
	}

	report := RenderBenchmarkReport(res)
	if len(report) == 0 {
		t.Errorf("Expected non-empty benchmark report")
	}
}

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSummaryAddCountsTotalAndFailedRequests(t *testing.T) {
	summary := newSummary()
	summary.add(result{latency: 200 * time.Millisecond})
	summary.add(result{err: errors.New("test request failure")})
	summary.add(result{latency: 100 * time.Millisecond})

	report := summary.report()

	if report.Total != 3 {
		t.Fatalf("report.Total = %d, want 3", report.Total)
	}
	if report.Failed != 1 {
		t.Fatalf("report.Failed = %d, want 1", report.Failed)
	}
	if report.Successful != 2 {
		t.Fatalf("report.Successful = %d, want 2", report.Successful)
	}
}

func TestSummaryReportIgnoresFailedRequestsForLatencyStats(t *testing.T) {
	summary := newSummary()
	summary.add(result{latency: 200 * time.Millisecond})
	summary.add(result{latency: 50 * time.Millisecond, err: errors.New("test request failure")})
	summary.add(result{latency: 100 * time.Millisecond})

	report := summary.report()

	if report.Min != 100*time.Millisecond {
		t.Fatalf("report.Min = %v, want %v", report.Min, 100*time.Millisecond)
	}
	if report.Max != 200*time.Millisecond {
		t.Fatalf("report.Max = %v, want %v", report.Max, 200*time.Millisecond)
	}
	if report.Avg != 150*time.Millisecond {
		t.Fatalf("report.Avg = %v, want %v", report.Avg, 150*time.Millisecond)
	}
}

func TestSummaryReportCalculatesMinMaxAndAverageLatency(t *testing.T) {
	summary := newSummary()
	summary.add(result{latency: 200 * time.Millisecond})
	summary.add(result{latency: 300 * time.Millisecond})
	summary.add(result{latency: 100 * time.Millisecond})

	report := summary.report()

	if report.Min != 100*time.Millisecond {
		t.Fatalf("report.Min = %v, want %v", report.Min, 100*time.Millisecond)
	}
	if report.Max != 300*time.Millisecond {
		t.Fatalf("report.Max = %v, want %v", report.Max, 300*time.Millisecond)
	}
	if report.Avg != 200*time.Millisecond {
		t.Fatalf("report.Avg = %v, want %v", report.Avg, 200*time.Millisecond)
	}
}

func TestSummaryReportCalculatesPercentilesFromSortedLatencies(t *testing.T) {
	summary := newSummary()
	summary.add(result{latency: 300 * time.Millisecond})
	summary.add(result{latency: 100 * time.Millisecond})
	summary.add(result{latency: 500 * time.Millisecond})
	summary.add(result{latency: 200 * time.Millisecond})
	summary.add(result{latency: 400 * time.Millisecond})

	report := summary.report()

	if report.P50 != 300*time.Millisecond {
		t.Fatalf("report.P50 = %v, want %v", report.P50, 300*time.Millisecond)
	}
	if report.P95 != 500*time.Millisecond {
		t.Fatalf("report.P95 = %v, want %v", report.P95, 500*time.Millisecond)
	}
	if report.P99 != 500*time.Millisecond {
		t.Fatalf("report.P99 = %v, want %v", report.P99, 500*time.Millisecond)
	}
}

func TestSummaryReportCountsNoSuccessfulRequestsWhenAllRequestsFail(t *testing.T) {
	summary := newSummary()
	summary.add(result{err: errors.New("first request failed")})
	summary.add(result{err: errors.New("second request failed")})

	report := summary.report()

	if report.Total != 2 {
		t.Fatalf("report.Total = %d, want 2", report.Total)
	}
	if report.Failed != 2 {
		t.Fatalf("report.Failed = %d, want 2", report.Failed)
	}
	if report.Successful != 0 {
		t.Fatalf("report.Successful = %d, want 0", report.Successful)
	}
}

func TestSummaryReportWriteToPrintsLatencyStats(t *testing.T) {
	report := summaryReport{
		Total:      3,
		Successful: 2,
		Failed:     1,
		Min:        100 * time.Millisecond,
		Max:        200 * time.Millisecond,
		Avg:        150 * time.Millisecond,
		P50:        100 * time.Millisecond,
		P95:        200 * time.Millisecond,
		P99:        200 * time.Millisecond,
	}
	var output bytes.Buffer

	_, err := report.WriteTo(&output)

	if err != nil {
		t.Fatalf("report.WriteTo() err = %v, want nil", err)
	}
	want := strings.Join([]string{
		"Total: 3",
		"Successful: 2",
		"Failed: 1",
		"Min: 100ms",
		"Max: 200ms",
		"Avg: 150ms",
		"P50: 100ms",
		"P95: 200ms",
		"P99: 200ms",
		"",
	}, "\n")
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestSummaryReportWriteToPrintsLatencyAsNotAvailableWhenAllRequestsFail(t *testing.T) {
	report := summaryReport{Total: 2, Failed: 2}
	var output bytes.Buffer

	_, err := report.WriteTo(&output)

	if err != nil {
		t.Fatalf("report.WriteTo() err = %v, want nil", err)
	}
	want := strings.Join([]string{
		"Total: 2",
		"Successful: 0",
		"Failed: 2",
		"Latency: n/a",
		"",
	}, "\n")
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

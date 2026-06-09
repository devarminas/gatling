package main

import (
	"bytes"
	"errors"
	"slices"
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

func TestSummaryReportCountsSuccessfulStatusDistribution(t *testing.T) {
	summary := newSummary()
	summary.add(result{statusCode: 200, latency: 100 * time.Millisecond})
	summary.add(result{statusCode: 201, latency: 100 * time.Millisecond})
	summary.add(result{statusCode: 200, latency: 100 * time.Millisecond})

	report := summary.report()

	want := []statusCodeCount{
		{StatusCode: 200, Count: 2},
		{StatusCode: 201, Count: 1},
	}
	if !slices.Equal(report.StatusCodes, want) {
		t.Fatalf("report.StatusCodes = %#v, want %#v", report.StatusCodes, want)
	}
}

func TestSummaryReportCountsFailedHTTPStatusAndError(t *testing.T) {
	summary := newSummary()
	summary.add(result{statusCode: 500, err: errors.New("read response body: broken pipe")})

	report := summary.report()

	if report.Failed != 1 {
		t.Fatalf("report.Failed = %d, want 1", report.Failed)
	}
	wantStatusCodes := []statusCodeCount{{StatusCode: 500, Count: 1}}
	if !slices.Equal(report.StatusCodes, wantStatusCodes) {
		t.Fatalf("report.StatusCodes = %#v, want %#v", report.StatusCodes, wantStatusCodes)
	}
	wantErrors := []errorCount{{Message: "read response body: broken pipe", Count: 1}}
	if !slices.Equal(report.Errors, wantErrors) {
		t.Fatalf("report.Errors = %#v, want %#v", report.Errors, wantErrors)
	}
}

func TestSummaryReportCountsRequestErrorWithoutStatusOnlyAsError(t *testing.T) {
	summary := newSummary()
	summary.add(result{err: errors.New("connection refused")})

	report := summary.report()

	if len(report.StatusCodes) != 0 {
		t.Fatalf("report.StatusCodes = %#v, want empty", report.StatusCodes)
	}
	wantErrors := []errorCount{{Message: "connection refused", Count: 1}}
	if !slices.Equal(report.Errors, wantErrors) {
		t.Fatalf("report.Errors = %#v, want %#v", report.Errors, wantErrors)
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

func TestSummaryReportWriteToPrintsStatusCodesAndErrorsDeterministically(t *testing.T) {
	report := summaryReport{
		Total:      5,
		Successful: 2,
		Failed:     3,
		Min:        100 * time.Millisecond,
		Max:        200 * time.Millisecond,
		Avg:        150 * time.Millisecond,
		P50:        100 * time.Millisecond,
		P95:        200 * time.Millisecond,
		P99:        200 * time.Millisecond,
		StatusCodes: []statusCodeCount{
			{StatusCode: 200, Count: 2},
			{StatusCode: 500, Count: 1},
		},
		Errors: []errorCount{
			{Message: "connection refused", Count: 2},
			{Message: "read response body: broken pipe", Count: 1},
		},
	}
	var output bytes.Buffer

	_, err := report.WriteTo(&output)

	if err != nil {
		t.Fatalf("report.WriteTo() err = %v, want nil", err)
	}
	want := strings.Join([]string{
		"Total: 5",
		"Successful: 2",
		"Failed: 3",
		"Min: 100ms",
		"Max: 200ms",
		"Avg: 150ms",
		"P50: 100ms",
		"P95: 200ms",
		"P99: 200ms",
		"Status Codes:",
		"  200: 2",
		"  500: 1",
		"Errors:",
		"  connection refused: 2",
		"  read response body: broken pipe: 1",
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

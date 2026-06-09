package main

import (
	"fmt"
	"io"
	"math"
	"slices"
	"time"
)

type summary struct {
	total       int
	failed      int
	sum         time.Duration
	latencies   []time.Duration
	statusCodes map[int]int
	errors      map[string]int
}

type statusCodeCount struct {
	StatusCode int
	Count      int
}

type errorCount struct {
	Message string
	Count   int
}

type summaryReport struct {
	Total       int
	Successful  int
	Failed      int
	Min         time.Duration
	Max         time.Duration
	Avg         time.Duration
	P50         time.Duration
	P95         time.Duration
	P99         time.Duration
	StatusCodes []statusCodeCount
	Errors      []errorCount
}

func newSummary() summary {
	return summary{}
}

func (s *summary) add(r result) {
	s.total++
	if r.statusCode > 0 {
		if s.statusCodes == nil {
			s.statusCodes = make(map[int]int)
		}
		s.statusCodes[r.statusCode]++
	}

	if r.err != nil {
		s.failed++
		if s.errors == nil {
			s.errors = make(map[string]int)
		}
		s.errors[r.err.Error()]++
		return
	}

	s.latencies = append(s.latencies, r.latency)
	s.sum += r.latency
}

func (s summary) report() summaryReport {
	sorted := append([]time.Duration(nil), s.latencies...)
	slices.Sort(sorted)

	return summaryReport{
		Total:       s.total,
		Successful:  len(sorted),
		Failed:      s.failed,
		Min:         min(sorted),
		Max:         max(sorted),
		Avg:         avg(s.sum, len(sorted)),
		P50:         percentile(sorted, 50),
		P95:         percentile(sorted, 95),
		P99:         percentile(sorted, 99),
		StatusCodes: statusCodeCounts(s.statusCodes),
		Errors:      errorCounts(s.errors),
	}
}

func (r summaryReport) WriteTo(w io.Writer) (int64, error) {
	totalWritten := int64(0)
	write := func(format string, args ...any) error {
		written, err := fmt.Fprintf(w, format, args...)
		totalWritten += int64(written)
		return err
	}

	if err := write("Total: %d\nSuccessful: %d\nFailed: %d\n", r.Total, r.Successful, r.Failed); err != nil {
		return totalWritten, err
	}

	if r.Successful == 0 {
		if err := write("Latency: n/a\n"); err != nil {
			return totalWritten, err
		}
	} else {
		if err := write(
			"Min: %v\nMax: %v\nAvg: %v\nP50: %v\nP95: %v\nP99: %v\n",
			r.Min,
			r.Max,
			r.Avg,
			r.P50,
			r.P95,
			r.P99,
		); err != nil {
			return totalWritten, err
		}
	}

	if len(r.StatusCodes) > 0 {
		if err := write("Status Codes:\n"); err != nil {
			return totalWritten, err
		}
		for _, statusCode := range r.StatusCodes {
			if err := write("  %d: %d\n", statusCode.StatusCode, statusCode.Count); err != nil {
				return totalWritten, err
			}
		}
	}

	if len(r.Errors) > 0 {
		if err := write("Errors:\n"); err != nil {
			return totalWritten, err
		}
		for _, error := range r.Errors {
			if err := write("  %s: %d\n", error.Message, error.Count); err != nil {
				return totalWritten, err
			}
		}
	}

	return totalWritten, nil
}

func statusCodeCounts(counts map[int]int) []statusCodeCount {
	statusCodes := make([]int, 0, len(counts))
	for statusCode := range counts {
		statusCodes = append(statusCodes, statusCode)
	}
	slices.Sort(statusCodes)

	result := make([]statusCodeCount, 0, len(statusCodes))
	for _, statusCode := range statusCodes {
		result = append(result, statusCodeCount{StatusCode: statusCode, Count: counts[statusCode]})
	}
	return result
}

func errorCounts(counts map[string]int) []errorCount {
	messages := make([]string, 0, len(counts))
	for message := range counts {
		messages = append(messages, message)
	}
	slices.Sort(messages)

	result := make([]errorCount, 0, len(messages))
	for _, message := range messages {
		result = append(result, errorCount{Message: message, Count: counts[message]})
	}
	return result
}

func avg(sum time.Duration, count int) time.Duration {
	if count == 0 {
		return 0
	}
	avgNanos := sum.Nanoseconds() / int64(count)
	return time.Duration(avgNanos)
}

func min(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	return latencies[0]
}

func max(latencies []time.Duration) time.Duration {
	length := len(latencies)
	if length == 0 {
		return 0
	}

	return latencies[length-1]
}

func percentile(latencies []time.Duration, p float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	i := int(math.Ceil((p/100)*float64(len(latencies)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(latencies) {
		i = len(latencies) - 1
	}

	return latencies[i]
}

package main

import (
	"fmt"
	"io"
	"math"
	"sort"
	"time"
)

type summary struct {
	total     int
	failed    int
	sum       time.Duration
	latencies []time.Duration
}

type summaryReport struct {
	Total      int
	Successful int
	Failed     int
	Min        time.Duration
	Max        time.Duration
	Avg        time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
}

func newSummary() summary {
	return summary{}
}

func (s *summary) add(r result) {
	s.total++
	if r.err != nil {
		s.failed++
		return
	}

	s.latencies = append(s.latencies, r.latency)
	s.sum += r.latency
}

func (s summary) report() summaryReport {
	sorted := append([]time.Duration(nil), s.latencies...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return summaryReport{
		Total:      s.total,
		Successful: len(sorted),
		Failed:     s.failed,
		Min:        min(sorted),
		Max:        max(sorted),
		Avg:        avg(s.sum, len(sorted)),
		P50:        percentile(sorted, 50),
		P95:        percentile(sorted, 95),
		P99:        percentile(sorted, 99),
	}
}

func (r summaryReport) WriteTo(w io.Writer) (int64, error) {
	written, err := fmt.Fprintf(w, "Total: %d\nSuccessful: %d\nFailed: %d\n", r.Total, r.Successful, r.Failed)
	totalWritten := int64(written)
	if err != nil {
		return totalWritten, err
	}

	if r.Successful == 0 {
		written, err = fmt.Fprint(w, "Latency: n/a\n")
		totalWritten += int64(written)
		return totalWritten, err
	}

	written, err = fmt.Fprintf(
		w,
		"Min: %v\nMax: %v\nAvg: %v\nP50: %v\nP95: %v\nP99: %v\n",
		r.Min,
		r.Max,
		r.Avg,
		r.P50,
		r.P95,
		r.P99,
	)
	totalWritten += int64(written)
	return totalWritten, err
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

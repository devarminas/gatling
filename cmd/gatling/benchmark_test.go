package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunConcurrentRecordsSuccessfulRequests(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("response body")),
		}, nil
	})}

	summary := runConcurrent(client, config{requests: 2, concurrency: 1, url: "http://example.test"})
	report := summary.report()

	if report.Total != 2 {
		t.Fatalf("report.Total = %d, want 2", report.Total)
	}
	if report.Successful != 2 {
		t.Fatalf("report.Successful = %d, want 2", report.Successful)
	}
	if report.Failed != 0 {
		t.Fatalf("report.Failed = %d, want 0", report.Failed)
	}
	if report.Min <= 0 {
		t.Fatalf("report.Min = %v, want > 0", report.Min)
	}
}

func TestRunRequestRecordsRequestError(t *testing.T) {
	wantErr := errors.New("request failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})}

	result := runRequest(client, config{url: "http://example.test"})

	if !errors.Is(result.err, wantErr) {
		t.Fatalf("result.err = %v, want %v", result.err, wantErr)
	}
	if result.statusCode != 0 {
		t.Fatalf("result.statusCode = %d, want 0", result.statusCode)
	}
}

func TestRunRequestRecordsBodyReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       errReaderCloser{readErr: wantErr},
		}, nil
	})}

	result := runRequest(client, config{url: "http://example.test"})

	if !errors.Is(result.err, wantErr) {
		t.Fatalf("result.err = %v, want %v", result.err, wantErr)
	}
	if result.statusCode != http.StatusAccepted {
		t.Fatalf("result.statusCode = %d, want %d", result.statusCode, http.StatusAccepted)
	}
}

func TestRunRequestRecordsBodyCloseError(t *testing.T) {
	wantErr := errors.New("close failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       errReaderCloser{closeErr: wantErr},
		}, nil
	})}

	result := runRequest(client, config{url: "http://example.test"})

	if !errors.Is(result.err, wantErr) {
		t.Fatalf("result.err = %v, want %v", result.err, wantErr)
	}
	if result.statusCode != http.StatusNoContent {
		t.Fatalf("result.statusCode = %d, want %d", result.statusCode, http.StatusNoContent)
	}
}

func TestRunConcurrentLimitsActiveRequests(t *testing.T) {
	entered := make(chan struct{}, 6)
	release := make(chan struct{})
	errors := make(chan string, 6)
	var active int32
	var maxActive int32

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		current := atomic.AddInt32(&active, 1)
		if current > 3 {
			errors <- "more than 3 active requests"
		}
		for {
			max := atomic.LoadInt32(&maxActive)
			if current <= max || atomic.CompareAndSwapInt32(&maxActive, max, current) {
				break
			}
		}

		entered <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("response body")),
		}, nil
	})}

	done := make(chan summary, 1)
	go func() {
		done <- runConcurrent(client, config{requests: 6, concurrency: 3, url: "http://example.test"})
	}()

	for range 3 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for 3 active requests")
		}
	}
	close(release)

	var got summary
	select {
	case got = <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runConcurrent")
	}

	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}

	report := got.report()
	if report.Total != 6 {
		t.Fatalf("report.Total = %d, want 6", report.Total)
	}
	if report.Successful != 6 {
		t.Fatalf("report.Successful = %d, want 6", report.Successful)
	}
	if maxActive > 3 {
		t.Fatalf("max active requests = %d, want <= 3", maxActive)
	}
	if maxActive != 3 {
		t.Fatalf("max active requests = %d, want 3", maxActive)
	}
}

func TestRunConcurrentDurationModeIgnoresRequestCount(t *testing.T) {
	var requests int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(&requests, 1)
		time.Sleep(time.Millisecond)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("response body")),
		}, nil
	})}

	summary := runConcurrent(client, config{requests: 1, concurrency: 1, duration: 100 * time.Millisecond, url: "http://example.test"})
	report := summary.report()

	if report.Total <= 1 {
		t.Fatalf("report.Total = %d, want > 1", report.Total)
	}
	if got := int32(report.Total); got != atomic.LoadInt32(&requests) {
		t.Fatalf("report.Total = %d, requests = %d", got, requests)
	}
}

func TestRunConcurrentDurationModeLimitsActiveRequestsAndStops(t *testing.T) {
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	errors := make(chan string, 3)
	var active int32
	var maxActive int32

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		current := atomic.AddInt32(&active, 1)
		if current > 3 {
			errors <- "more than 3 active requests"
		}
		for {
			max := atomic.LoadInt32(&maxActive)
			if current <= max || atomic.CompareAndSwapInt32(&maxActive, max, current) {
				break
			}
		}

		entered <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("response body")),
		}, nil
	})}

	done := make(chan summary, 1)
	go func() {
		done <- runConcurrent(client, config{requests: 1, concurrency: 3, duration: 100 * time.Millisecond, url: "http://example.test"})
	}()

	for range 3 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for 3 active requests")
		}
	}

	time.Sleep(150 * time.Millisecond)
	close(release)

	var got summary
	select {
	case got = <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runConcurrent")
	}

	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}

	report := got.report()
	if report.Total != 3 {
		t.Fatalf("report.Total = %d, want 3", report.Total)
	}
	if report.Successful != 3 {
		t.Fatalf("report.Successful = %d, want 3", report.Successful)
	}
	if maxActive != 3 {
		t.Fatalf("max active requests = %d, want 3", maxActive)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReaderCloser struct {
	readErr  error
	closeErr error
}

func (e errReaderCloser) Read([]byte) (int, error) {
	if e.readErr != nil {
		return 0, e.readErr
	}
	return 0, io.EOF
}

func (e errReaderCloser) Close() error {
	return e.closeErr
}

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

func TestRunSequentialSuccess(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("response body")),
		}, nil
	})}

	results := runConcurrent(client, config{requests: 2, concurrency: 1, url: "http://example.test"})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, got := range results {
		if got.statusCode != http.StatusCreated {
			t.Fatalf("results[%d].statusCode = %d, want %d", i, got.statusCode, http.StatusCreated)
		}
		if got.err != nil {
			t.Fatalf("results[%d].err = %v, want nil", i, got.err)
		}
		if got.latency <= 0 {
			t.Fatalf("results[%d].latency = %v, want > 0", i, got.latency)
		}
	}
}

func TestRunSequentialRecordsRequestError(t *testing.T) {
	wantErr := errors.New("request failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})}

	results := runConcurrent(client, config{requests: 1, concurrency: 1, url: "http://example.test"})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !errors.Is(results[0].err, wantErr) {
		t.Fatalf("results[0].err = %v, want %v", results[0].err, wantErr)
	}
	if results[0].statusCode != 0 {
		t.Fatalf("results[0].statusCode = %d, want 0", results[0].statusCode)
	}
}

func TestRunSequentialRecordsBodyReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       errReaderCloser{readErr: wantErr},
		}, nil
	})}

	results := runConcurrent(client, config{requests: 1, concurrency: 1, url: "http://example.test"})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !errors.Is(results[0].err, wantErr) {
		t.Fatalf("results[0].err = %v, want %v", results[0].err, wantErr)
	}
	if results[0].statusCode != http.StatusAccepted {
		t.Fatalf("results[0].statusCode = %d, want %d", results[0].statusCode, http.StatusAccepted)
	}
}

func TestRunSequentialRecordsBodyCloseError(t *testing.T) {
	wantErr := errors.New("close failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       errReaderCloser{closeErr: wantErr},
		}, nil
	})}

	results := runConcurrent(client, config{requests: 1, concurrency: 1, url: "http://example.test"})

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !errors.Is(results[0].err, wantErr) {
		t.Fatalf("results[0].err = %v, want %v", results[0].err, wantErr)
	}
	if results[0].statusCode != http.StatusNoContent {
		t.Fatalf("results[0].statusCode = %d, want %d", results[0].statusCode, http.StatusNoContent)
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

	done := make(chan []result, 1)
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

	var results []result
	select {
	case results = <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runConcurrent")
	}

	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}

	if len(results) != 6 {
		t.Fatalf("len(results) = %d, want 6", len(results))
	}
	if maxActive > 3 {
		t.Fatalf("max active requests = %d, want <= 3", maxActive)
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

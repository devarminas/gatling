package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-h"}, &stdout, &stderr, http.DefaultClient)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gatling [options] <url>") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunRequiresURL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(nil, &stdout, &stderr, http.DefaultClient)

	if code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: gatling [options] <url>") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunReportsInvalidConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-n", "0", "http://example.test"}, &stdout, &stderr, http.DefaultClient)

	if code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got, want := strings.TrimSpace(stderr.String()), "-n must be greater than 0, got 0"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRunSuccessfulRequest(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})}

	code := run([]string{"-n", "1", "http://example.test"}, &stdout, &stderr, client)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[{200 ") {
		t.Fatalf("stdout = %q, want result with status 200", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReportsOutputWriteError(t *testing.T) {
	var stderr bytes.Buffer
	wantErr := errors.New("stdout failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})}

	code := run([]string{"-n", "1", "http://example.test"}, errWriter{err: wantErr}, &stderr, client)

	if code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if got, want := strings.TrimSpace(stderr.String()), "write output: stdout failed"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

type errWriter struct {
	err error
}

func (w errWriter) Write([]byte) (int, error) {
	return 0, w.err
}

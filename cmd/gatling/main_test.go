package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  config
		wantErr string
	}{
		{
			name: "valid config",
			config: config{
				requests:    10,
				concurrency: 2,
				url:         "https://example.com",
			},
		},
		{
			name: "zero requests",
			config: config{
				requests:    0,
				concurrency: 1,
				url:         "https://example.com",
			},
			wantErr: "-n must be greater than 0, got 0",
		},
		{
			name: "zero concurrency",
			config: config{
				requests:    10,
				concurrency: 0,
				url:         "https://example.com",
			},
			wantErr: "-c must be greater than 0, got 0",
		},
		{
			name: "concurrency exceeds requests",
			config: config{
				requests:    5,
				concurrency: 6,
				url:         "https://example.com",
			},
			wantErr: "-c (6) cannot exceed -n (5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantCode        int
		wantStdout      string
		wantStderr      []string
		wantNoStdout    bool
		wantNoStderr    bool
		wantNoDuplicate string
	}{
		{
			name:         "valid args",
			args:         []string{"-n", "3", "-c", "2", "https://example.com"},
			wantCode:     0,
			wantStdout:   "{3 2 https://example.com}\n",
			wantNoStderr: true,
		},
		{
			name:         "help",
			args:         []string{"-h"},
			wantCode:     0,
			wantStderr:   []string{"Usage: gatling [options] <url>", "Options:", "-n int", "-c int"},
			wantNoStdout: true,
		},
		{
			name:         "missing url",
			args:         nil,
			wantCode:     2,
			wantStderr:   []string{"Usage: gatling [options] <url>", "Options:", "-n int", "-c int"},
			wantNoStdout: true,
		},
		{
			name:            "invalid flag",
			args:            []string{"-x"},
			wantCode:        2,
			wantStderr:      []string{"flag provided but not defined: -x", "Usage: gatling [options] <url>"},
			wantNoStdout:    true,
			wantNoDuplicate: "flag provided but not defined: -x",
		},
		{
			name:         "invalid requests",
			args:         []string{"-n", "0", "https://example.com"},
			wantCode:     2,
			wantStderr:   []string{"-n must be greater than 0, got 0"},
			wantNoStdout: true,
		},
		{
			name:         "invalid concurrency",
			args:         []string{"-c", "0", "https://example.com"},
			wantCode:     2,
			wantStderr:   []string{"-c must be greater than 0, got 0"},
			wantNoStdout: true,
		},
		{
			name:         "concurrency exceeds requests",
			args:         []string{"-n", "2", "-c", "3", "https://example.com"},
			wantCode:     2,
			wantStderr:   []string{"-c (3) cannot exceed -n (2)"},
			wantNoStdout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("expected exit code %d, got %d", tt.wantCode, code)
			}

			gotStdout := stdout.String()
			gotStderr := stderr.String()

			if tt.wantNoStdout && gotStdout != "" {
				t.Fatalf("expected empty stdout, got %q", gotStdout)
			}
			if tt.wantNoStderr && gotStderr != "" {
				t.Fatalf("expected empty stderr, got %q", gotStderr)
			}
			if tt.wantStdout != "" && gotStdout != tt.wantStdout {
				t.Fatalf("expected stdout %q, got %q", tt.wantStdout, gotStdout)
			}

			for _, want := range tt.wantStderr {
				if !strings.Contains(gotStderr, want) {
					t.Fatalf("expected stderr to contain %q, got %q", want, gotStderr)
				}
			}
			if tt.wantNoDuplicate != "" && strings.Count(gotStderr, tt.wantNoDuplicate) != 1 {
				t.Fatalf("expected stderr to contain %q once, got %q", tt.wantNoDuplicate, gotStderr)
			}
		})
	}
}

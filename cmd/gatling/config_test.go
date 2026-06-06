package main

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config
		wantErr string
	}{
		{
			name: "valid",
			cfg:  config{requests: 10, concurrency: 2, url: "https://example.com"},
		},
		{
			name:    "requests must be positive",
			cfg:     config{requests: 0, concurrency: 1, url: "https://example.com"},
			wantErr: "-n must be greater than 0, got 0",
		},
		{
			name:    "concurrency must be positive",
			cfg:     config{requests: 1, concurrency: 0, url: "https://example.com"},
			wantErr: "-c must be greater than 0, got 0",
		},
		{
			name:    "concurrency cannot exceed requests",
			cfg:     config{requests: 2, concurrency: 3, url: "https://example.com"},
			wantErr: "-c (3) cannot exceed -n (2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validate() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("validate() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

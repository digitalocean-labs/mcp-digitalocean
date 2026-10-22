package main

import (
	"strings"
	"testing"
)

func TestCheckAnonymousServices(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		wantErr  bool
	}{
		{name: "docs only", services: []string{"docs"}},
		{name: "all services", services: nil, wantErr: true},
		{name: "account scoped", services: []string{"droplets"}, wantErr: true},
		{name: "docs plus account scoped", services: []string{"docs", "droplets"}, wantErr: true},
		// The registry resolves service names case-sensitively and does not trim,
		// so these never start. Reject them here rather than reporting that
		// anonymous access is enabled and failing later.
		{name: "wrong case", services: []string{"Docs"}, wantErr: true},
		{name: "untrimmed", services: []string{" docs"}, wantErr: true},
		{name: "trailing comma leaves an empty name", services: []string{"docs", ""}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAnonymousServices(tt.services)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkAnonymousServices(%v) error = %v, wantErr %v", tt.services, err, tt.wantErr)
			}
		})
	}
}

func TestCheckAnonymousServicesErrorNamesEmptyEntry(t *testing.T) {
	err := checkAnonymousServices([]string{"docs", ""})
	if err == nil {
		t.Fatal("expected an error for an empty service name")
	}
	if !strings.Contains(err.Error(), `""`) {
		t.Fatalf("error should quote the empty name so it is visible, got %q", err)
	}
}

func TestInvalidBoolEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  string
	}{
		{name: "unset", want: ""},
		{name: "valid", value: "true", set: true, want: ""},
		{name: "valid numeric", value: "1", set: true, want: ""},
		{name: "unparseable is reported verbatim", value: "yes", set: true, want: "yes"},
		{name: "on is not a bool", value: "on", set: true, want: "on"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("ALLOW_ANONYMOUS", tt.value)
			}
			if got := invalidBoolEnv("ALLOW_ANONYMOUS"); got != tt.want {
				t.Fatalf("invalidBoolEnv(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset keeps fallback", want: false},
		{name: "true", value: "true", set: true, want: true},
		{name: "uppercase", value: "TRUE", set: true, want: true},
		{name: "numeric", value: "1", set: true, want: true},
		{name: "padded", value: " true ", set: true, want: true},
		{name: "false", value: "false", set: true, want: false},
		{name: "unparseable falls back", value: "yes please", set: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("ALLOW_ANONYMOUS", tt.value)
			}
			if got := getEnvBool("ALLOW_ANONYMOUS", false); got != tt.want {
				t.Fatalf("getEnvBool(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

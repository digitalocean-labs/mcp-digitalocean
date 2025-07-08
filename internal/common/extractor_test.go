package common

import (
	"testing"
)

func TestExtractNumericIDFromURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantID  int64
		wantErr bool
	}{
		{
			name:    "valid numeric ID",
			uri:     "droplet://12345",
			wantID:  12345,
			wantErr: false,
		},
		{
			name:    "valid numeric ID with leading zeros",
			uri:     "droplet://00123",
			wantID:  123,
			wantErr: false,
		},
		{
			name:    "invalid URI format - missing separator",
			uri:     "droplet12345",
			wantErr: true,
		},
		{
			name:    "invalid URI format - empty ID",
			uri:     "droplet://",
			wantErr: true,
		},
		{
			name:    "invalid URI format - non-numeric ID",
			uri:     "droplet://abc123",
			wantErr: true,
		},
		{
			name:    "invalid URI format - multiple separators",
			uri:     "droplet://123://456",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := ExtractNumericIDFromURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractNumericIDFromURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotID != tt.wantID {
				t.Errorf("ExtractNumericIDFromURI() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestExtractStringIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantUUID string
		wantErr  bool
	}{
		{
			name:     "valid UUID",
			uri:      "droplet://550e8400-e29b-41d4-a716-446655440000",
			wantUUID: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:  false,
		},
		{
			name:     "valid string ID",
			uri:      "droplet://test-id",
			wantUUID: "test-id",
			wantErr:  false,
		},
		{
			name:    "invalid URI format - missing separator",
			uri:     "droplettest-id",
			wantErr: true,
		},
		{
			name:     "invalid URI format - empty ID",
			uri:      "droplet://",
			wantUUID: "",
			wantErr:  false,
		},
		{
			name:     "invalid URI format - multiple separators",
			uri:      "droplet://id://123",
			wantUUID: "",
			wantErr:  true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUUID, err := ExtractStringIDFromURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractStringIDFromURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotUUID != tt.wantUUID {
				t.Errorf("ExtractStringIDFromURI() = %v, want %v", gotUUID, tt.wantUUID)
			}
		})
	}
}

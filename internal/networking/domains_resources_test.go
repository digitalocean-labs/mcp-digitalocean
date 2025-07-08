package networking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupDomainResourceWithMock(domains *MockDomainsService) *DomainMCPResource {
	client := &godo.Client{}
	client.Domains = domains
	return NewDomainMCPResource(client)
}

func TestDomainMCPResource_handleGetDomainResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDomain := &godo.Domain{
		Name:     "example.com",
		TTL:      1800,
		ZoneFile: "zonefile",
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name: "Successful get",
			uri:  "domains://example.com",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Get(gomock.Any(), "example.com").
					Return(testDomain, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "domains://fail.com",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Get(gomock.Any(), "fail.com").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "domains123",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			resource := setupDomainResourceWithMock(mockDomains)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetDomainResource(context.Background(), req)
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp, 1)
			content, ok := resp[0].(mcp.TextResourceContents)
			require.True(t, ok)
			require.Equal(t, tc.uri, content.URI)
			require.Equal(t, "application/json", content.MIMEType)
			var outDomain godo.Domain
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outDomain))
			require.Equal(t, testDomain.Name, outDomain.Name)
		})
	}
}

func TestDomainMCPResource_handleGetDomainRecordResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRecord := &godo.DomainRecord{
		ID:   123,
		Type: "A",
		Name: "www",
		Data: "203.0.113.20",
	}
	tests := []struct {
		name        string
		uri         string
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name: "Successful get record",
			uri:  "domains://example.com/records/123",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Record(gomock.Any(), "example.com", 123).
					Return(testRecord, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			uri:  "domains://fail.com/records/456",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Record(gomock.Any(), "fail.com", 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Invalid URI format",
			uri:         "domains://badformat",
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Non-numeric record ID",
			uri:         "domains://example.com/records/abc",
			mockSetup:   nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			resource := setupDomainResourceWithMock(mockDomains)
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: tc.uri,
				},
			}
			resp, err := resource.handleGetDomainRecordResource(context.Background(), req)
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp, 1)
			content, ok := resp[0].(mcp.TextResourceContents)
			require.True(t, ok)
			require.Equal(t, tc.uri, content.URI)
			require.Equal(t, "application/json", content.MIMEType)
			var outRecord godo.DomainRecord
			require.NoError(t, json.Unmarshal([]byte(content.Text), &outRecord))
			require.Equal(t, testRecord.ID, outRecord.ID)
		})
	}
}

// Additional edge case: test extractDomainAndRecordFromURI directly
func Test_extractDomainAndRecordFromURI(t *testing.T) {
	tests := []struct {
		uri         string
		wantDomain  string
		wantRecord  int
		expectError bool
	}{
		{"example.com/records/123", "example.com", 123, false},
		{"bad/records/abc", "", 0, true},
		{"badformat", "", 0, true},
		{"", "", 0, true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("uri=%s", tc.uri), func(t *testing.T) {
			domain, record, err := extractDomainAndRecordFromURI(tc.uri)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantDomain, domain)
				require.Equal(t, tc.wantRecord, record)
			}
		})
	}
}

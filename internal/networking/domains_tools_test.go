package networking

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupDomainsToolWithMock(domains *MockDomainsService) *DomainsTool {
	client := &godo.Client{}
	client.Domains = domains
	return NewDomainsTool(client)
}

func TestDomainsTool_createDomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDomain := &godo.Domain{
		Name:     "example.com",
		TTL:      1800,
		ZoneFile: "zonefile",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":      "example.com",
				"IPAddress": "203.0.113.10",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.DomainCreateRequest{
						Name:      "example.com",
						IPAddress: "203.0.113.10",
					}).
					Return(testDomain, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":      "fail.com",
				"IPAddress": "203.0.113.20",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.DomainCreateRequest{
						Name:      "fail.com",
						IPAddress: "203.0.113.20",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createDomain(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outDomain godo.Domain
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outDomain))
			require.Equal(t, testDomain.Name, outDomain.Name)
		})
	}
}

func TestDomainsTool_deleteDomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDomainsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{"Name": "example.com"},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Delete(gomock.Any(), "example.com").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Domain deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{"Name": "fail.com"},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Delete(gomock.Any(), "fail.com").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteDomain(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestDomainsTool_createRecord(t *testing.T) {
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
		args        map[string]any
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name: "Successful create record",
			args: map[string]any{
				"Domain": "example.com",
				"Type":   "A",
				"Name":   "www",
				"Data":   "203.0.113.20",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					CreateRecord(gomock.Any(), "example.com", &godo.DomainRecordEditRequest{
						Type: "A",
						Name: "www",
						Data: "203.0.113.20",
					}).
					Return(testRecord, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Domain": "fail.com",
				"Type":   "TXT",
				"Name":   "test",
				"Data":   "fail",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					CreateRecord(gomock.Any(), "fail.com", &godo.DomainRecordEditRequest{
						Type: "TXT",
						Name: "test",
						Data: "fail",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createRecord(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outRecord godo.DomainRecord
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outRecord))
			require.Equal(t, testRecord.ID, outRecord.ID)
		})
	}
}

func TestDomainsTool_deleteRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDomainsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete record",
			args: map[string]any{
				"Domain":   "example.com",
				"RecordID": float64(123),
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					DeleteRecord(gomock.Any(), "example.com", 123).
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "Record deleted successfully",
		},
		{
			name: "API error",
			args: map[string]any{
				"Domain":   "fail.com",
				"RecordID": float64(456),
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					DeleteRecord(gomock.Any(), "fail.com", 456).
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteRecord(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestDomainsTool_editRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRecord := &godo.DomainRecord{
		ID:   789,
		Type: "CNAME",
		Name: "blog",
		Data: "host.example.com",
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name: "Successful edit record",
			args: map[string]any{
				"Domain":   "example.com",
				"RecordID": float64(789),
				"Type":     "CNAME",
				"Name":     "blog",
				"Data":     "host.example.com",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					EditRecord(gomock.Any(), "example.com", 789, &godo.DomainRecordEditRequest{
						Type: "CNAME",
						Name: "blog",
						Data: "host.example.com",
					}).
					Return(testRecord, nil, nil).
					Times(1)
			},
		},
		{
			name: "API error",
			args: map[string]any{
				"Domain":   "fail.com",
				"RecordID": float64(999),
				"Type":     "A",
				"Name":     "fail",
				"Data":     "fail",
			},
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					EditRecord(gomock.Any(), "fail.com", 999, &godo.DomainRecordEditRequest{
						Type: "A",
						Name: "fail",
						Data: "fail",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.editRecord(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var outRecord godo.DomainRecord
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outRecord))
			require.Equal(t, testRecord.ID, outRecord.ID)
		})
	}
}

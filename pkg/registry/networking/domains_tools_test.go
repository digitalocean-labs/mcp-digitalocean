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
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Domains: domains}, nil
	}

	return NewDomainsTool(client)
}

func TestDomainsTool_getDomain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDomain := &godo.Domain{
		Name:     "example.com",
		TTL:      1800,
		ZoneFile: "zonefile",
	}
	tests := []struct {
		name        string
		domain      string
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name:   "Successful get",
			domain: "example.com",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Get(gomock.Any(), "example.com").
					Return(testDomain, nil, nil).
					Times(1)
			},
		},
		{
			name:   "API error",
			domain: "fail.com",
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Get(gomock.Any(), "fail.com").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing domain argument",
			domain:      "",
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
			tool := setupDomainsToolWithMock(mockDomains)
			args := map[string]any{}
			if tc.name != "Missing domain argument" {
				args["Name"] = tc.domain
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getDomain(context.Background(), req)
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

func TestDomainsTool_listDomains(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testDomains := []godo.Domain{
		{Name: "example.com"},
		{Name: "test.com"},
	}
	tests := []struct {
		name        string
		page        float64
		perPage     float64
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name:    "Successful list",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testDomains, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testDomains, nil, nil).
					Times(1)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDomains := NewMockDomainsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockDomains)
			}
			tool := setupDomainsToolWithMock(mockDomains)
			args := map[string]any{}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listDomains(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outDomains []godo.Domain
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outDomains))
			require.GreaterOrEqual(t, len(outDomains), 1)
		})
	}
}

func TestDomainsTool_getDomainRecord(t *testing.T) {
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
		domain      string
		recordID    float64
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name:     "Successful get record",
			domain:   "example.com",
			recordID: 123,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Record(gomock.Any(), "example.com", 123).
					Return(testRecord, nil, nil).
					Times(1)
			},
		},
		{
			name:     "API error",
			domain:   "fail.com",
			recordID: 456,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Record(gomock.Any(), "fail.com", 456).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "Missing domain argument",
			domain:      "",
			recordID:    123,
			mockSetup:   nil,
			expectError: true,
		},
		{
			name:        "Missing recordID argument",
			domain:      "example.com",
			recordID:    0,
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
			tool := setupDomainsToolWithMock(mockDomains)
			args := map[string]any{}
			if tc.domain != "" {
				args["Domain"] = tc.domain
			}
			if tc.name != "Missing recordID argument" {
				args["RecordID"] = tc.recordID
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.getDomainRecord(context.Background(), req)
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

func TestDomainsTool_listDomainRecords(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRecords := []godo.DomainRecord{
		{ID: 1, Type: "A", Name: "www", Data: "1.2.3.4"},
		{ID: 2, Type: "CNAME", Name: "blog", Data: "host.example.com"},
	}
	tests := []struct {
		name        string
		domain      string
		page        float64
		perPage     float64
		mockSetup   func(*MockDomainsService)
		expectError bool
	}{
		{
			name:    "Successful list records",
			domain:  "example.com",
			page:    2,
			perPage: 1,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Records(gomock.Any(), "example.com", &godo.ListOptions{Page: 2, PerPage: 1}).
					Return(testRecords, nil, nil).
					Times(1)
			},
		},
		{
			name:    "API error",
			domain:  "fail.com",
			page:    1,
			perPage: 2,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Records(gomock.Any(), "fail.com", &godo.ListOptions{Page: 1, PerPage: 2}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:    "Default pagination",
			domain:  "example.com",
			page:    0,
			perPage: 0,
			mockSetup: func(m *MockDomainsService) {
				m.EXPECT().
					Records(gomock.Any(), "example.com", &godo.ListOptions{Page: 1, PerPage: 20}).
					Return(testRecords, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing domain argument",
			domain:      "",
			page:        1,
			perPage:     1,
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
			tool := setupDomainsToolWithMock(mockDomains)
			args := map[string]any{}
			if tc.domain != "" {
				args["Domain"] = tc.domain
			}
			if tc.page != 0 {
				args["Page"] = tc.page
			}
			if tc.perPage != 0 {
				args["PerPage"] = tc.perPage
			}
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
			resp, err := tool.listDomainRecords(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			var outRecords []godo.DomainRecord
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &outRecords))
			require.GreaterOrEqual(t, len(outRecords), 1)
		})
	}
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

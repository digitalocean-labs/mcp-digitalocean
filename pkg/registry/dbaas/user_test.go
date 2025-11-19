package dbaas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"mcp-digitalocean/pkg/registry/dbaas/mocks"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
)

func newUserToolWithMock(t *testing.T) (*UserTool, *mocks.MockDatabasesService, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockDatabasesService(ctrl)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Databases: mockSvc,
		}, nil
	}

	return &UserTool{client: client}, mockSvc, ctrl
}

func getTextContent(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestUserTool_getUser(t *testing.T) {
	tool, mockSvc, ctrl := newUserToolWithMock(t)
	defer ctrl.Finish()
	ctx := context.Background()

	dbUser := &godo.DatabaseUser{Name: "testuser"}
	mockSvc.EXPECT().GetUser(ctx, "cid", "testuser").Return(dbUser, nil, nil)

	// Success
	res, err := tool.getUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "testuser"}}})
	assert.NoError(t, err)
	assert.Contains(t, getTextContent(res), "testuser")

	// Missing Cluster ID
	res, err = tool.getUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"user": "testuser"}}})
	assert.NoError(t, err)
	assert.Equal(t, "Cluster id is required", getTextContent(res))

	// Missing user
	res, err = tool.getUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid"}}})
	assert.NoError(t, err)
	assert.Equal(t, "User name is required", getTextContent(res))

	// API error
	errApi := errors.New("api fail")
	mockSvc.EXPECT().GetUser(ctx, "cid", "failuser").Return(nil, nil, errApi)
	res, err = tool.getUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "failuser"}}})
	assert.NoError(t, err)
	assert.Contains(t, getTextContent(res), "api error")
}

func TestUserTool_listUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		users := []godo.DatabaseUser{{Name: "u1"}, {Name: "u2"}}
		mockSvc.EXPECT().ListUsers(ctx, "cid", (*godo.ListOptions)(nil)).Return(users, nil, nil)
		res, err := tool.listUsers(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "u1")
		assert.Contains(t, getTextContent(res), "u2")
	})

	t.Run("pagination", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		users := []godo.DatabaseUser{{Name: "u1"}, {Name: "u2"}}
		opts := &godo.ListOptions{Page: 2, PerPage: 5}
		mockSvc.EXPECT().ListUsers(ctx, "cid", opts).Return(users, nil, nil)
		res, err := tool.listUsers(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "page": "2", "per_page": 5}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "u1")
	})

	t.Run("missing cluster ID", func(t *testing.T) {
		tool, _, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		res, err := tool.listUsers(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}})
		assert.NoError(t, err)
		assert.Equal(t, "Cluster id is required", getTextContent(res))
	})

	t.Run("api error", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		errApi := errors.New("api fail")
		mockSvc.EXPECT().ListUsers(ctx, "cid", (*godo.ListOptions)(nil)).Return(nil, nil, errApi)
		res, err := tool.listUsers(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "api error")
	})
}

func TestUserTool_createUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		dbUser := &godo.DatabaseUser{Name: "newuser"}
		mockSvc.EXPECT().CreateUser(ctx, "cid", gomock.Any()).Return(dbUser, nil, nil)

		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"id":   "cid",
				"name": "newuser",
			}},
		})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "newuser")
	})

	t.Run("with mysql_auth_plugin", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		dbUser := &godo.DatabaseUser{Name: "pluginuser"}
		mockSvc.EXPECT().CreateUser(ctx, "cid", mock.MatchedBy(func(req *godo.DatabaseCreateUserRequest) bool {
			return req.MySQLSettings != nil && req.MySQLSettings.AuthPlugin == "mysql_native_password"
		})).Return(dbUser, nil, nil)

		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"id":                "cid",
				"name":              "pluginuser",
				"mysql_auth_plugin": "mysql_native_password",
			}},
		})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "pluginuser")
	})

	t.Run("with settings object", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		dbUser := &godo.DatabaseUser{Name: "settingsuser"}
		mockSvc.EXPECT().CreateUser(ctx, "cid", gomock.Any()).Return(dbUser, nil, nil)

		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"id":   "cid",
				"name": "settingsuser",
				"settings": map[string]any{
					"acl": []map[string]any{
						{"id": "acl1", "permission": "read", "topic": "topic1"},
					},
				},
			}},
		})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "settingsuser")
	})

	t.Run("invalid settings (non-object type)", func(t *testing.T) {
		// Use a real mock, but do not set any expectation, so handler can reach settings parsing
		tool, _, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		args := map[string]any{
			"id":       "cid",
			"name":     "broken",
			"settings": "notjson",
		}
		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: args},
		})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "Invalid settings object")
	})

	t.Run("missing cluster ID", func(t *testing.T) {
		client := func(ctx context.Context) (*godo.Client, error) {
			return &godo.Client{}, nil
		}

		tool := &UserTool{client: client}
		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"name": "newuser",
			}},
		})
		assert.NoError(t, err)
		assert.Equal(t, "Cluster id is required", getTextContent(res))
	})

	t.Run("missing name", func(t *testing.T) {
		client := func(ctx context.Context) (*godo.Client, error) {
			return &godo.Client{}, nil
		}

		tool := &UserTool{client: client}
		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"id": "cid",
			}},
		})
		assert.NoError(t, err)
		assert.Equal(t, "User name is required", getTextContent(res))
	})

	t.Run("api error", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		mockSvc.EXPECT().CreateUser(ctx, "cid", gomock.Any()).Return(nil, nil, fmt.Errorf("api fail"))

		res, err := tool.createUser(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{Arguments: map[string]any{
				"id":   "cid",
				"name": "failuser",
			}},
		})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "api error")
	})
}

func TestUserTool_updateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		dbUser := &godo.DatabaseUser{Name: "updateduser"}
		mockSvc.EXPECT().UpdateUser(ctx, "cid", "updateduser", gomock.Any()).Return(dbUser, nil, nil)
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "updateduser"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "updateduser")
	})

	t.Run("with settings_json", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		dbUser := &godo.DatabaseUser{Name: "updateduser"}
		settings := godo.DatabaseUserSettings{ACL: []*godo.KafkaACL{{ID: "acl2", Permission: "write"}}}
		settingsBytes, _ := json.Marshal(settings)
		var settingsMap map[string]any
		_ = json.Unmarshal(settingsBytes, &settingsMap)
		mockSvc.EXPECT().UpdateUser(ctx, "cid", "updateduser", mock.MatchedBy(func(req *godo.DatabaseUpdateUserRequest) bool {
			return req.Settings != nil && len(req.Settings.ACL) > 0 && req.Settings.ACL[0].ID == "acl2"
		})).Return(dbUser, nil, nil)
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "updateduser", "settings": settingsMap}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "updateduser")
	})

	t.Run("invalid settings_json", func(t *testing.T) {
		tool := &UserTool{client: nil}
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "updateduser", "settings": "notjson"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "Invalid settings object")
	})

	t.Run("missing cluster ID", func(t *testing.T) {
		tool := &UserTool{client: nil}
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"user": "updateduser"}}})
		assert.NoError(t, err)
		assert.Equal(t, "Cluster id is required", getTextContent(res))
	})

	t.Run("missing user", func(t *testing.T) {
		tool := &UserTool{client: nil}
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid"}}})
		assert.NoError(t, err)
		assert.Equal(t, "User name is required", getTextContent(res))
	})

	t.Run("api error", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		errApi := errors.New("api fail")
		mockSvc.EXPECT().UpdateUser(ctx, "cid", "failuser", gomock.Any()).Return(nil, nil, errApi)
		res, err := tool.updateUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "failuser"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "api error")
	})
}

func TestUserTool_deleteUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		mockSvc.EXPECT().DeleteUser(ctx, "cid", "deluser").Return((*godo.Response)(nil), nil)
		res, err := tool.deleteUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "deluser"}}})
		assert.NoError(t, err)
		assert.Equal(t, "User deleted successfully", getTextContent(res))
	})

	t.Run("missing cluster ID", func(t *testing.T) {
		tool := &UserTool{client: nil}
		res, err := tool.deleteUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"user": "deluser"}}})
		assert.NoError(t, err)
		assert.Equal(t, "Cluster id is required", getTextContent(res))
	})

	t.Run("missing user", func(t *testing.T) {
		tool := &UserTool{client: nil}
		res, err := tool.deleteUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid"}}})
		assert.NoError(t, err)
		assert.Equal(t, "User name is required", getTextContent(res))
	})

	t.Run("api error", func(t *testing.T) {
		tool, mockSvc, ctrl := newUserToolWithMock(t)
		defer ctrl.Finish()
		errApi := errors.New("api fail")
		mockSvc.EXPECT().DeleteUser(ctx, "cid", "failuser").Return((*godo.Response)(nil), errApi)
		res, err := tool.deleteUser(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": "cid", "user": "failuser"}}})
		assert.NoError(t, err)
		assert.Contains(t, getTextContent(res), "api error")
	})
}

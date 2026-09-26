package registry

import (
	"io"
	"log/slog"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

// TestProjectToolsFollowDropletsAndAccounts locks the registration rule:
// project tools are exposed when droplets or accounts is enabled, once when
// both are, and not when neither is. The server stores tools by name, so a
// second registration overwrites rather than duplicating; the assertion is
// that each project tool is present exactly once.
func TestProjectToolsFollowDropletsAndAccounts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	projectTools := []string{"project-list", "project-get", "project-get-default", "project-create"}

	tests := []struct {
		name     string
		services []string
		want     bool
	}{
		{name: "droplets", services: []string{"droplets"}, want: true},
		{name: "accounts", services: []string{"accounts"}, want: true},
		{name: "droplets and accounts", services: []string{"droplets", "accounts"}, want: true},
		{name: "apps", services: []string{"apps"}, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svr := server.NewMCPServer("test", "test")
			if err := Register(logger, svr, annotationTestClient, tc.services...); err != nil {
				t.Fatalf("Register: %v", err)
			}

			tools := svr.ListTools()
			var got int
			for _, name := range projectTools {
				if _, ok := tools[name]; ok {
					got++
				}
			}
			want := 0
			if tc.want {
				want = len(projectTools)
			}
			if got != want {
				t.Fatalf("project tools present = %d, want %d", got, want)
			}
		})
	}
}

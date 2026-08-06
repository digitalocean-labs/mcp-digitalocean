package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type AppSummary struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Url       string          `json:"url"`
	ProjectID string          `json:"project_id,omitempty"`
	Region    *godo.AppRegion `json:"region"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Tier      string          `json:"tier"`
}

func toAppSummary(app *godo.App) *AppSummary {
	return &AppSummary{
		ID:        app.GetID(),
		Name:      app.GetSpec().GetName(),
		Url:       app.GetLiveURL(),
		ProjectID: app.GetProjectID(),
		Region:    app.GetRegion(),
		CreatedAt: app.GetCreatedAt(),
		UpdatedAt: app.GetUpdatedAt(),
		Tier:      app.GetTierSlug(),
	}
}

func main() {
	token := os.Getenv("DIGITALOCEAN_API_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr, "Error: DIGITALOCEAN_API_TOKEN environment variable is not set\n")
		fmt.Fprintf(os.Stderr, "Please set it with: export DIGITALOCEAN_API_TOKEN=your_token\n")
		os.Exit(1)
	}

	// Clean token (remove quotes if present)
	cleanToken := strings.Trim(strings.TrimSpace(token), "'")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cleanToken})
	oauthClient := oauth2.NewClient(context.Background(), ts)

	client := godo.NewClient(oauthClient)

	ctx := context.Background()
	
	// List all apps
	apps, _, err := client.Apps.List(ctx, &godo.ListOptions{Page: 1, PerPage: 200})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing apps: %v\n", err)
		os.Exit(1)
	}

	if len(apps) == 0 {
		fmt.Println("No apps found in your DigitalOcean account.")
		return
	}

	// Convert to summaries
	summaries := make([]*AppSummary, len(apps))
	for i, app := range apps {
		summaries[i] = toAppSummary(app)
	}

	// Output as JSON
	appsJSON, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(appsJSON))
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"mcp-digitalocean/internal/apps"

	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"
)

//go:generate go run .

// We generate the JSON schema from godo structs for AppCreateRequest and AppUpdateRequest.
// This is necessary since we need to pass the AppSpec to the mcp tool as a raw argument.
// Ideally, we shouldn't have to copy the godo files around. However, it's currently not possible to without preserving the struct comments.
func main() {
	if err := generateSchemas(); err != nil {
		log.Fatalf("Failed to generate schemas: %v", err)
	}
}

func generateSchemas() error {
	reflect := jsonschema.Reflector{
		BaseSchemaID:               "",
		Anonymous:                  true,
		AssignAnchor:               false,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
		ExpandedStruct:             true,
		FieldNameTag:               "",
	}

	if err := reflect.AddGoComments("github.com/digitalocean/godo", "./"); err != nil {
		return fmt.Errorf("failed to add Go comments: %w", err)
	}

	// Generate app create schema
	if err := generateAppCreateSchema(reflect); err != nil {
		return fmt.Errorf("failed to generate app create schema: %w", err)
	}

	// Generate app update schema
	if err := generateAppUpdateSchema(reflect); err != nil {
		return fmt.Errorf("failed to generate app update schema: %w", err)
	}

	return nil
}

func generateAppCreateSchema(reflect jsonschema.Reflector) error {
	createSchema, err := reflect.Reflect(&godo.AppCreateRequest{}).MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal app create schema: %w", err)
	}

	var createSchemaJSON bytes.Buffer
	if err := json.Indent(&createSchemaJSON, createSchema, "", "  "); err != nil {
		return fmt.Errorf("failed to indent JSON: %w", err)
	}

	if err := os.WriteFile("./app-create-schema.json", createSchemaJSON.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	fmt.Println("Schema successfully written to app-create-schema.json")
	return nil
}

func generateAppUpdateSchema(reflect jsonschema.Reflector) error {
	updateSchema, err := reflect.Reflect(&apps.AppUpdate{}).MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal app update schema: %w", err)
	}

	var updateSchemaJSON bytes.Buffer
	if err := json.Indent(&updateSchemaJSON, updateSchema, "", "  "); err != nil {
		return fmt.Errorf("failed to indent JSON: %w", err)
	}

	if err := os.WriteFile("./app-update-schema.json", updateSchemaJSON.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	fmt.Println("Update schema successfully written to app-update-schema.json")
	return nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mcp-digitalocean/pkg/registry/apps"
	"os"

	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"
)

//go:generate go run .

// We generate the JSON schema from godo structs for AppCreateRequest and AppUpdateRequest.
// This is necessary since we need to pass the AppSpec to the mcp tool as a raw argument.
// Ideally, we shouldn't have to copy the godo files around. However, it's currently not possible to without preserving the struct comments.
func main() {
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

	err := reflect.AddGoComments("github.com/digitalocean/godo", "./")
	if err != nil {
		panic(fmt.Errorf("failed to add Go comments: %w", err))
	}

	createSchema, err := reflect.Reflect(&godo.AppCreateRequest{}).MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal app create schema: %w", err))
	}

	var createSchemaJSON bytes.Buffer
	if err := json.Indent(&createSchemaJSON, createSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// now write the schema to a file
	err = os.WriteFile("./app-create-schema.json", createSchemaJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	fmt.Println("Schema successfully written to app_create_schema.json")

	// Generate schema for AppUpdateRequest
	updateSchema, err := reflect.Reflect(&apps.AppUpdate{}).MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal app update schema: %w", err))
	}

	// Prettify the JSON
	var updateSchemaJSON bytes.Buffer
	if err := json.Indent(&updateSchemaJSON, updateSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// Write the schema to a file
	err = os.WriteFile("./app-update-schema.json", updateSchemaJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	fmt.Println("Update schema successfully written to app_update_schema.json")
}

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"mcp-digitalocean/internal/apps"
	
	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"
	// "github.com/wk8/go-ordered-map/v2" // added for ordered map
)

//go:generate go run .

// We generate the JSON schema from godo structs for AppCreateRequest and AppUpdateRequest.
// This is necessary since we need to pass the AppSpec to the mcp tool as a raw argument.
// Ideally, we shouldn't have to copy the godo files around. However, it's currently not possible to without preserving the struct comments.
// modified the function to a direct way to fix the issue for schema 2020-12 problems and add the logic that intercepts the generated schema and changes its version to draft-07.
func main() {
	reflect := jsonschema.Reflector{
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
	}
	err := reflect.AddGoComments("github.com/digitalocean/godo", "./")
	if err != nil {
		panic(fmt.Errorf("failed to add Go comments: %w", err))
	}

	// Generate schema for AppCreateRequest 
	createSchemaObject := reflect.Reflect(&godo.AppCreateRequest{})
	
	createSchemaBytes, err := json.Marshal(createSchemaObject)
	if err != nil {
		panic(fmt.Errorf("failed to marshal app create schema: %w", err))
	}

	var createSchemaMap map[string]interface{}
	if err := json.Unmarshal(createSchemaBytes, &createSchemaMap); err != nil {
		panic(fmt.Errorf("failed to unmarshal create schema to map: %w", err))
	}
	
	createSchemaMap["$schema"] = "http://json-schema.org/draft-07/schema#"

	finalCreateSchema, err := json.MarshalIndent(createSchemaMap, "", "  ")
	if err != nil {
		panic(fmt.Errorf("failed to marshal final create schema: %w", err))
	}
	
	// Write the schema to a file
	err = os.WriteFile("./app-create-schema.draft-07.json", finalCreateSchema, 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}
	fmt.Println("Draft-07 schema successfully written to app-create-schema.draft-07.json")


	// Generate schema for AppUpdateRequest 
	updateSchemaObject := reflect.Reflect(&apps.AppUpdate{})
	
	updateSchemaBytes, err := json.Marshal(updateSchemaObject)
	if err != nil {
		panic(fmt.Errorf("failed to marshal app update schema: %w", err))
	}

	var updateSchemaMap map[string]interface{}
	if err := json.Unmarshal(updateSchemaBytes, &updateSchemaMap); err != nil {
		panic(fmt.Errorf("failed to unmarshal update schema to map: %w", err))
	}
	
	updateSchemaMap["$schema"] = "http://json-schema.org/draft-07/schema#"
	
	finalUpdateSchema, err := json.MarshalIndent(updateSchemaMap, "", "  ")
	if err != nil {
		panic(fmt.Errorf("failed to marshal final update schema: %w", err))
	}

	err = os.WriteFile("./app-update-schema.draft-07.json", finalUpdateSchema, 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}
	fmt.Println("Draft-07 update schema successfully written to app-update-schema.draft-07.json")
}
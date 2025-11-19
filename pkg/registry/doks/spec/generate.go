package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"
)

//go:generate go run .

// We generate the JSON schema from godo structs for CreateClusterRequest and CreateNodePoolRequest.
// This is necessary since we need to pass the NodePoolSpecs to the mcp tool as a raw argument.
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

	createSchema, err := reflect.Reflect(&godo.KubernetesClusterCreateRequest{}).MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal cluster create schema: %w", err))
	}

	var createSchemaJSON bytes.Buffer
	if err := json.Indent(&createSchemaJSON, createSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// now write the schema to a file
	err = os.WriteFile("./cluster-create-schema.json", createSchemaJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	// Modify the cluster create schema to add minItems constraints
	clusterSchema := reflect.Reflect(&godo.KubernetesClusterCreateRequest{})

	// Add minItems constraint to node_pools array (cluster must have at least 1 node pool)
	if nodePoolsProperty, exists := clusterSchema.Properties.Get("node_pools"); exists {
		minItems := uint64(1)
		nodePoolsProperty.MinItems = &minItems

		// Also add minItems constraint to nested taints field within node pools
		if nodePoolsProperty.Items != nil && nodePoolsProperty.Items.Properties != nil {
			if taintsProperty, taintsExists := nodePoolsProperty.Items.Properties.Get("taints"); taintsExists {
				taintsMinItems := uint64(1)
				taintsProperty.MinItems = &taintsMinItems
				nodePoolsProperty.Items.Properties.Set("taints", taintsProperty)
			}
		}

		clusterSchema.Properties.Set("node_pools", nodePoolsProperty)
	}

	// Add minItems constraint to control_plane_firewall.allowed_addresses when present
	if firewallProperty, exists := clusterSchema.Properties.Get("control_plane_firewall"); exists {
		if firewallProperty.Properties != nil {
			if allowedAddressesProperty, addressesExists := firewallProperty.Properties.Get("allowed_addresses"); addressesExists {
				minItems := uint64(1)
				allowedAddressesProperty.MinItems = &minItems
				firewallProperty.Properties.Set("allowed_addresses", allowedAddressesProperty)
			}
		}
		clusterSchema.Properties.Set("control_plane_firewall", firewallProperty)
	}

	// Add minItems constraint to cluster_autoscaler_configuration.expanders when present
	if autoscalerProperty, exists := clusterSchema.Properties.Get("cluster_autoscaler_configuration"); exists {
		if autoscalerProperty.Properties != nil {
			if expandersProperty, expandersExists := autoscalerProperty.Properties.Get("expanders"); expandersExists {
				minItems := uint64(1)
				expandersProperty.MinItems = &minItems
				autoscalerProperty.Properties.Set("expanders", expandersProperty)
			}
		}
		clusterSchema.Properties.Set("cluster_autoscaler_configuration", autoscalerProperty)
	}

	// Re-marshal the modified cluster schema
	modifiedClusterSchema, err := clusterSchema.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal modified cluster create schema: %w", err))
	}

	var modifiedClusterSchemaJSON bytes.Buffer
	if err := json.Indent(&modifiedClusterSchemaJSON, modifiedClusterSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent modified cluster JSON: %w", err))
	}

	// Write the modified schema to file
	err = os.WriteFile("./cluster-create-schema.json", modifiedClusterSchemaJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write modified cluster schema to file: %w", err))
	}

	fmt.Println("Schema successfully written to cluster-create-schema.json")

	// Generate wrapped auxiliary schema for KubernetesNodePoolCreateRequest
	aux := struct {
		ClusterID             string                               `json:"cluster_id"`
		NodePoolCreateRequest godo.KubernetesNodePoolCreateRequest `json:"node_pool_create_request"`
	}{
		ClusterID: "",
		NodePoolCreateRequest: godo.KubernetesNodePoolCreateRequest{
			// Initialize with zero values
			Name:      "",
			Size:      "",
			Count:     0,
			Tags:      []string{},
			Labels:    map[string]string{},
			Taints:    []godo.Taint{},
			AutoScale: false,
			MinNodes:  0,
			MaxNodes:  0,
		},
	}

	reflector := &jsonschema.Reflector{}
	schema := reflector.Reflect(&aux)

	// Modify the taints field to require non-empty array or omission
	// Navigate to the KubernetesNodePoolCreateRequest definition
	if defs, ok := schema.Definitions["KubernetesNodePoolCreateRequest"]; ok {
		if taintsProperty, exists := defs.Properties.Get("taints"); exists {
			// Set minItems to 1 to ensure the array is non-empty when present
			minItems := uint64(1)
			taintsProperty.MinItems = &minItems
			defs.Properties.Set("taints", taintsProperty)
		}
	}

	npCreateSchema, err := schema.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal node pool create schema: %w", err))
	}

	// Prettify the JSON
	var npCreateSchemaJSON bytes.Buffer
	if err := json.Indent(&npCreateSchemaJSON, npCreateSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// Write the schema to a file
	err = os.WriteFile("./node-pool-create-schema.json", npCreateSchemaJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	fmt.Println("Schema successfully written to node-pool-create-schema.json")
}

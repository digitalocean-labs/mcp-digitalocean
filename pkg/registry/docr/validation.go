package docr

import (
	"fmt"
	"regexp"

	"github.com/mark3labs/mcp-go/mcp"
)

// Same rule the DOCR API enforces on registry names.
const registryNameMaxLength = 63

var registryNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

// validateRegistryName checks the name before it is used in an API request
// path. godo interpolates it unescaped, so a value like "../droplets/123"
// would otherwise be normalised by the HTTP client into a request to a
// different endpoint.
func validateRegistryName(name string) error {
	if name == "" {
		return fmt.Errorf("RegistryName is required")
	}
	if len(name) > registryNameMaxLength {
		return fmt.Errorf("RegistryName must be at most %d characters", registryNameMaxLength)
	}
	if !registryNameRegexp.MatchString(name) {
		return fmt.Errorf("RegistryName must contain only lowercase letters, digits and hyphens, start with a letter and end with a letter or digit")
	}
	return nil
}

// registryNameArg extracts and validates the RegistryName argument. On failure
// it returns a tool error result that the handler should return as-is.
func registryNameArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	name, _ := req.GetArguments()["RegistryName"].(string)
	if err := validateRegistryName(name); err != nil {
		return "", mcp.NewToolResultError(err.Error())
	}
	return name, nil
}

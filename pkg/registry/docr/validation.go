package docr

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

// These values are interpolated into API request paths unescaped, so they are
// checked against the same rules the DOCR API enforces before use.
const (
	registryNameMaxLength   = 63
	repositoryNameMaxLength = 255
)

var (
	registryNameRegexp   = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	repositoryNameRegexp = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$`)
	tagRegexp            = regexp.MustCompile(`^[\w][\w.-]{0,127}$`)
	digestRegexp         = regexp.MustCompile(`^[a-z0-9]+(?:[.+_-][a-z0-9]+)*:[a-zA-Z0-9=_-]+$`)
)

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

func validateRepositoryName(name string) error {
	if name == "" {
		return fmt.Errorf("Repository is required")
	}
	if len(name) > repositoryNameMaxLength {
		return fmt.Errorf("Repository must be at most %d characters", repositoryNameMaxLength)
	}
	if !repositoryNameRegexp.MatchString(name) {
		return fmt.Errorf("Repository must be a valid repository name: lowercase letters, digits and separators (., _, -), with path components separated by /")
	}
	return nil
}

func validateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("Tag is required")
	}
	if !tagRegexp.MatchString(tag) {
		return fmt.Errorf("Tag must be a valid image tag: letters, digits, underscores, periods and hyphens, not starting with a period or hyphen, at most 128 characters")
	}
	return nil
}

func validateDigest(digest string) error {
	if digest == "" {
		return fmt.Errorf("Digest is required")
	}
	if !digestRegexp.MatchString(digest) {
		return fmt.Errorf("Digest must be a valid manifest digest, for example sha256:<hex>")
	}
	return nil
}

func validateGarbageCollectionUUID(id string) error {
	if id == "" {
		return fmt.Errorf("GarbageCollectionUUID is required")
	}
	if err := uuid.Validate(id); err != nil {
		return fmt.Errorf("GarbageCollectionUUID must be a valid UUID")
	}
	return nil
}

// validatedArg reads a string argument and validates it; on failure it returns
// a tool error result for the handler to return as-is.
func validatedArg(req mcp.CallToolRequest, key string, validate func(string) error) (string, *mcp.CallToolResult) {
	value, _ := req.GetArguments()[key].(string)
	if err := validate(value); err != nil {
		return "", mcp.NewToolResultError(err.Error())
	}
	return value, nil
}

func registryNameArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	return validatedArg(req, "RegistryName", validateRegistryName)
}

func repositoryArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	return validatedArg(req, "Repository", validateRepositoryName)
}

func tagArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	return validatedArg(req, "Tag", validateTag)
}

func digestArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	return validatedArg(req, "Digest", validateDigest)
}

func garbageCollectionUUIDArg(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	return validatedArg(req, "GarbageCollectionUUID", validateGarbageCollectionUUID)
}

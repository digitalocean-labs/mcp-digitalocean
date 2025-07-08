package common

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractNumericIDFromURI extracts numeric ID from the uri
// Example: droplet://12345 returns 12345, usually the format for extraction is
// {resourceType}://{numericID} which returns the numeric ID as an int64
func ExtractNumericIDFromURI(uri string) (int64, error) {
	split := strings.Split(uri, "://")
	if len(split) != 2 {
		return 0, fmt.Errorf("invalid uri format")
	}

	id, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid uri format")
	}

	return id, nil
}

// ExtractStringIDFromURI extracts UUID or string from the uri
// Example: droplet://abc returns abc, usually the format for extraction is
// {resourceType}://{uuid} which returns the uuid as a string
func ExtractStringIDFromURI(uri string) (string, error) {
	split := strings.Split(uri, "://")
	if len(split) != 2 {
		return "", fmt.Errorf("invalid uri format")
	}

	uuid := split[1]
	return uuid, nil
}

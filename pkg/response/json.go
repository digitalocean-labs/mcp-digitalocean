package response

import (
	"encoding/json"
)

// CompactJSON returns minified JSON without indentation.
// This reduces response size significantly compared to json.MarshalIndent.
func CompactJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

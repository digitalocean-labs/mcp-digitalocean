package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "simple object",
			input:    map[string]interface{}{"id": 123, "name": "test"},
			expected: `{"id":123,"name":"test"}`,
			wantErr:  false,
		},
		{
			name:     "array",
			input:    []int{1, 2, 3},
			expected: `[1,2,3]`,
			wantErr:  false,
		},
		{
			name:     "nested object",
			input:    map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			expected: `{"outer":{"inner":"value"}}`,
			wantErr:  false,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: `null`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompactJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCompactJSON_ReducesSize(t *testing.T) {
	// Create a sample object similar to what we might get from DigitalOcean API
	data := map[string]interface{}{
		"id":     12345,
		"name":   "test-droplet",
		"status": "active",
		"region": map[string]interface{}{
			"name": "New York 3",
			"slug": "nyc3",
		},
		"size": map[string]interface{}{
			"slug":   "s-1vcpu-1gb",
			"memory": 1024,
			"vcpus":  1,
		},
	}

	// Compact JSON
	compact, err := CompactJSON(data)
	assert.NoError(t, err)

	// Regular indented JSON (what we're replacing)
	indented, err := json.MarshalIndent(data, "", "  ")
	assert.NoError(t, err)

	// Verify compact version is smaller
	assert.Less(t, len(compact), len(indented), "Compact JSON should be smaller than indented")

	// Verify they contain the same data when unmarshaled
	var compactData, indentedData map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(compact), &compactData))
	assert.NoError(t, json.Unmarshal(indented, &indentedData))
	assert.Equal(t, compactData, indentedData, "Data should be identical")
}

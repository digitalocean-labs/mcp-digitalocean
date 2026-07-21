package genaicustommodels

import (
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestCustomModelFromGodo_ErrorMessage(t *testing.T) {
	model := customModelFromGodo(&godo.CustomModel{
		Uuid:         "55555555-5555-5555-5555-555555555555",
		Name:         "team/failed-model",
		Status:       godo.CustomModelStatusFailed,
		ErrorMessage: "Repository not found or access denied",
	})

	require.NotNil(t, model)
	require.Equal(t, "Repository not found or access denied", model.ErrorMessage)
	require.Equal(t, CustomModelStatusFailed, model.Status)
}

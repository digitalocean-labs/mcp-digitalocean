package genaicustommodels

import (
	"encoding/json"
	"time"
)

// CustomModelStatus represents the status of a custom model.
type CustomModelStatus string

const (
	CustomModelStatusUnspecified CustomModelStatus = "STATUS_UNSPECIFIED"
	CustomModelStatusImporting   CustomModelStatus = "STATUS_IMPORTING"
	CustomModelStatusReady       CustomModelStatus = "STATUS_READY"
	CustomModelStatusFailed      CustomModelStatus = "STATUS_FAILED"
	CustomModelStatusDeleted     CustomModelStatus = "STATUS_DELETED"
)

// CustomModelSourceType represents the source type for a custom model.
type CustomModelSourceType string

const (
	CustomModelSourceTypeUnspecified CustomModelSourceType = "SOURCE_TYPE_UNSPECIFIED"
	CustomModelSourceTypeHuggingFace CustomModelSourceType = "SOURCE_TYPE_HUGGINGFACE"
	CustomModelSourceTypeSpaces      CustomModelSourceType = "SOURCE_TYPE_SPACES_BUCKET"
	CustomModelSourceTypeSDKUpload   CustomModelSourceType = "SOURCE_TYPE_SDK_UPLOAD"
	CustomModelSourceTypeFineTuning  CustomModelSourceType = "SOURCE_TYPE_FINE_TUNING"
)

// CustomModelAccessType represents the access type for a model source.
type CustomModelAccessType string

const (
	CustomModelAccessTypeUnspecified CustomModelAccessType = "ACCESS_TYPE_UNSPECIFIED"
	CustomModelAccessTypePublic      CustomModelAccessType = "ACCESS_TYPE_PUBLIC"
	CustomModelAccessTypePrivate     CustomModelAccessType = "ACCESS_TYPE_PRIVATE"
	CustomModelAccessTypeGated       CustomModelAccessType = "ACCESS_TYPE_GATED"
)

// CustomModelSourceRef describes where the model files come from.
type CustomModelSourceRef struct {
	// HuggingFace fields
	RepoID     string                `json:"repo_id,omitempty"`
	CommitSHA  string                `json:"commit_sha,omitempty"`
	AccessType CustomModelAccessType `json:"access_type,omitempty"`
	HFToken    string                `json:"hf_token,omitempty"`
	// Spaces Bucket fields
	Bucket string `json:"bucket,omitempty"`
	Region string `json:"region,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

// CustomModelTags wraps a list of tag strings.
type CustomModelTags struct {
	Tags []string `json:"tags,omitempty"`
}

// CustomModelDeploymentEndpoints holds endpoint FQDNs for a deployment.
type CustomModelDeploymentEndpoints struct {
	PublicEndpointFQDN  string `json:"public_endpoint_fqdn,omitempty"`
	PrivateEndpointFQDN string `json:"private_endpoint_fqdn,omitempty"`
}

// CustomModelActiveDeployment represents a deployment running this model.
type CustomModelActiveDeployment struct {
	ID         string                          `json:"id"`
	Name       string                          `json:"name"`
	RegionSlug string                          `json:"region_slug,omitempty"`
	State      string                          `json:"state,omitempty"`
	Endpoints  *CustomModelDeploymentEndpoints `json:"endpoints,omitempty"`
	CreatedAt  *time.Time                      `json:"created_at,omitempty"`
	UpdatedAt  *time.Time                      `json:"updated_at,omitempty"`
}

// CustomModel represents a custom model in the GenAI platform.
type CustomModel struct {
	UUID                 string                         `json:"uuid"`
	Name                 string                         `json:"name"`
	Description          string                         `json:"description,omitempty"`
	Status               CustomModelStatus              `json:"status"`
	Architecture         string                         `json:"architecture,omitempty"`
	SourceType           CustomModelSourceType          `json:"source_type,omitempty"`
	SourceRef            *CustomModelSourceRef          `json:"source_ref,omitempty"`
	TotalSizeBytes       json.Number                    `json:"total_size_bytes,omitempty"`
	FileCount            json.Number                    `json:"file_count,omitempty"`
	License              string                         `json:"license,omitempty"`
	Tags                 *CustomModelTags               `json:"tags,omitempty"`
	CreatedAt            *time.Time                     `json:"created_at,omitempty"`
	UpdatedAt            *time.Time                     `json:"updated_at,omitempty"`
	ActiveDeployments    []*CustomModelActiveDeployment `json:"active_deployments,omitempty"`
	ContextLength        json.Number                    `json:"context_length,omitempty"`
	CostEstimatePerMonth json.Number                    `json:"cost_estimate_per_month,omitempty"`
	InputModalities      []string                       `json:"input_modalities,omitempty"`
	OutputModalities     []string                       `json:"output_modalities,omitempty"`
	Parameters           json.Number                    `json:"parameters,omitempty"`
	TeamID               json.Number                    `json:"team_id,omitempty"`
	ConfigJSON           map[string]interface{}         `json:"config_json,omitempty"`
	StorageRegion        string                         `json:"storage_region,omitempty"`
}

// PaginationLinks holds pagination links from the API.
type PaginationLinks struct {
	Pages *PaginationPages `json:"pages,omitempty"`
}

// PaginationPages holds the page URLs.
type PaginationPages struct {
	First string `json:"first,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Last  string `json:"last,omitempty"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page,omitempty"`
	Pages int `json:"pages,omitempty"`
}

// ListCustomModelsOutput is the response from listing custom models.
type ListCustomModelsOutput struct {
	Models       []*CustomModel   `json:"models"`
	Links        *PaginationLinks `json:"links,omitempty"`
	Meta         *PaginationMeta  `json:"meta,omitempty"`
	MaxThreshold int              `json:"max_threshold,omitempty"`
}

// ImportCustomModelInput is the request body for importing a custom model.
type ImportCustomModelInput struct {
	Name                     string                `json:"name"`
	SourceType               CustomModelSourceType `json:"source_type"`
	SourceRef                CustomModelSourceRef  `json:"source_ref"`
	Description              string                `json:"description,omitempty"`
	PreferredGPURegion       string                `json:"preferred_gpu_region,omitempty"`
	AcceptTermsAndConditions bool                  `json:"accept_terms_and_conditions"`
	Tags                     *CustomModelTags      `json:"tags,omitempty"`
}

// ImportJob represents the status of a model import job.
type ImportJob struct {
	UUID       string      `json:"uuid"`
	Status     string      `json:"status"`
	FilesTotal json.Number `json:"files_total"`
	FilesDone  json.Number `json:"files_done"`
	BytesTotal json.Number `json:"bytes_total"`
	BytesDone  json.Number `json:"bytes_done"`
	CreatedAt  *time.Time  `json:"created_at,omitempty"`
}

// ValidationStep represents a validation check during import.
type ValidationStep struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// ImportCustomModelOutput is the response from importing a custom model.
type ImportCustomModelOutput struct {
	Model           *CustomModel      `json:"model"`
	ImportJob       *ImportJob        `json:"import_job,omitempty"`
	ValidationSteps []*ValidationStep `json:"validation_steps,omitempty"`
	Error           string            `json:"error,omitempty"`
}

// UpdateCustomModelMetadataInput is the request body for updating model metadata.
type UpdateCustomModelMetadataInput struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Tags        *CustomModelTags `json:"tags,omitempty"`
}

// UpdateCustomModelMetadataOutput is the response from updating model metadata.
type UpdateCustomModelMetadataOutput struct {
	Model *CustomModel `json:"model"`
}

// DeleteCustomModelOutput is the response from deleting a custom model.
type DeleteCustomModelOutput struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// GetCustomModelOutput is the response from getting a single custom model.
type GetCustomModelOutput struct {
	Model *CustomModel `json:"model"`
}

// CatalogSearchRow is one catalog model row in unified search output.
type CatalogSearchRow struct {
	UUID             string   `json:"uuid"`
	Name             string   `json:"name"`
	Source           string   `json:"source"`
	Provider         string   `json:"provider,omitempty"`
	Type             string   `json:"type,omitempty"`
	ContextWindow    string   `json:"context_window,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
}

// CustomSearchRow is one custom model row in unified search output.
type CustomSearchRow struct {
	UUID             string   `json:"uuid"`
	Name             string   `json:"name"`
	Source           string   `json:"source"`
	Status           string   `json:"status"`
	Architecture     string   `json:"architecture,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
}

// UnifiedSearchResponse is the response from the unified model search tool.
type UnifiedSearchResponse struct {
	Query         string              `json:"query"`
	CatalogModels []CatalogSearchRow  `json:"catalog_models"`
	CustomModels  []CustomSearchRow   `json:"custom_models"`
	Counts        UnifiedSearchCounts `json:"counts"`
}

// UnifiedSearchCounts holds per-source result counts.
type UnifiedSearchCounts struct {
	Catalog int `json:"catalog"`
	Custom  int `json:"custom"`
	Total   int `json:"total"`
}

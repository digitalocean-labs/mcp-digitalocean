package genaicustommodels

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
)

func customModelFromGodo(m *godo.CustomModel) *CustomModel {
	if m == nil {
		return nil
	}
	out := &CustomModel{
		UUID:                 m.Uuid,
		Name:                 m.Name,
		Description:          m.Description,
		Status:               CustomModelStatus(m.Status),
		Architecture:         m.Architecture,
		SourceType:           CustomModelSourceType(m.SourceType),
		TotalSizeBytes:       jsonNumberFromString(m.TotalSizeBytes),
		FileCount:            jsonNumberFromInt(m.FileCount),
		License:              m.License,
		CreatedAt:            timestampToTime(m.CreatedAt),
		UpdatedAt:            timestampToTime(m.UpdatedAt),
		ContextLength:        jsonNumberFromInt(m.ContextLength),
		CostEstimatePerMonth: jsonNumberFromInt(m.CostEstimatePerMonth),
		InputModalities:      append([]string(nil), m.InputModalities...),
		OutputModalities:     append([]string(nil), m.OutputModalities...),
		Parameters:           jsonNumberFromString(m.Parameters),
		TeamID:               jsonNumberFromString(m.TeamId),
		ConfigJSON:           m.ConfigJson,
		StorageRegion:        m.StorageRegion,
	}
	if m.SourceRef != nil {
		out.SourceRef = &CustomModelSourceRef{
			RepoID:     m.SourceRef.RepoId,
			CommitSHA:  m.SourceRef.CommitSha,
			AccessType: CustomModelAccessType(m.SourceRef.AccessType),
			HFToken:    m.SourceRef.HfToken,
			Bucket:     m.SourceRef.Bucket,
			Region:     m.SourceRef.Region,
			Prefix:     m.SourceRef.Prefix,
		}
	}
	if m.Tags != nil {
		out.Tags = &CustomModelTags{Tags: append([]string(nil), m.Tags.Tags...)}
	}
	if len(m.ActiveDeployments) > 0 {
		out.ActiveDeployments = make([]*CustomModelActiveDeployment, 0, len(m.ActiveDeployments))
		for _, d := range m.ActiveDeployments {
			if d == nil {
				continue
			}
			dep := &CustomModelActiveDeployment{
				ID:         d.Id,
				Name:       d.Name,
				RegionSlug: d.RegionSlug,
				State:      d.State,
			}
			if d.Endpoints != nil {
				dep.Endpoints = &CustomModelDeploymentEndpoints{
					PublicEndpointFQDN:  d.Endpoints.PublicEndpointFqdn,
					PrivateEndpointFQDN: d.Endpoints.PrivateEndpointFqdn,
				}
			}
			out.ActiveDeployments = append(out.ActiveDeployments, dep)
		}
	}
	return out
}

func customModelsFromGodo(models []*godo.CustomModel) []*CustomModel {
	if len(models) == 0 {
		return nil
	}
	out := make([]*CustomModel, 0, len(models))
	for _, m := range models {
		out = append(out, customModelFromGodo(m))
	}
	return out
}

func importRequestToGodo(in *ImportCustomModelInput) *godo.CustomModelImportRequest {
	req := &godo.CustomModelImportRequest{
		Name:                     in.Name,
		SourceType:               godo.CustomModelSourceType(in.SourceType),
		Description:              in.Description,
		PreferredGpuRegion:       in.PreferredGPURegion,
		AcceptTermsAndConditions: in.AcceptTermsAndConditions,
	}
	req.SourceRef = &godo.CustomModelSourceRef{
		RepoId:     in.SourceRef.RepoID,
		CommitSha:  in.SourceRef.CommitSHA,
		AccessType: godo.CustomModelSourceRefAccessType(in.SourceRef.AccessType),
		HfToken:    in.SourceRef.HFToken,
		Bucket:     in.SourceRef.Bucket,
		Region:     in.SourceRef.Region,
		Prefix:     in.SourceRef.Prefix,
	}
	if in.Tags != nil {
		req.Tags = &godo.CustomModelTags{Tags: append([]string(nil), in.Tags.Tags...)}
	}
	return req
}

func importResponseFromGodo(out *godo.CustomModelImportResponse) *ImportCustomModelOutput {
	if out == nil {
		return nil
	}
	resp := &ImportCustomModelOutput{
		Model: customModelFromGodo(out.Model),
		Error: out.Error,
	}
	if out.ImportJob != nil {
		resp.ImportJob = &ImportJob{
			UUID:       out.ImportJob.Uuid,
			Status:     out.ImportJob.Status,
			FilesTotal: jsonNumberFromInt(out.ImportJob.FilesTotal),
			FilesDone:  jsonNumberFromInt(out.ImportJob.FilesDone),
			BytesTotal: jsonNumberFromString(out.ImportJob.BytesTotal),
			BytesDone:  jsonNumberFromString(out.ImportJob.BytesDone),
			CreatedAt:  timestampToTime(out.ImportJob.CreatedAt),
		}
	}
	if len(out.ValidationSteps) > 0 {
		resp.ValidationSteps = make([]*ValidationStep, 0, len(out.ValidationSteps))
		for _, s := range out.ValidationSteps {
			if s == nil {
				continue
			}
			resp.ValidationSteps = append(resp.ValidationSteps, &ValidationStep{
				Name:   s.Name,
				Passed: s.Passed,
				Error:  s.Error,
			})
		}
	}
	return resp
}

func metadataUpdateToGodo(in *UpdateCustomModelMetadataInput) *godo.CustomModelMetadataUpdateRequest {
	req := &godo.CustomModelMetadataUpdateRequest{}
	if in.Name != nil {
		req.Name = *in.Name
	}
	if in.Description != nil {
		req.Description = *in.Description
	}
	if in.Tags != nil {
		req.Tags = &godo.CustomModelTags{Tags: append([]string(nil), in.Tags.Tags...)}
	}
	return req
}

func deleteResponseFromGodo(out *godo.CustomModelDeleteResponse) *DeleteCustomModelOutput {
	if out == nil {
		return &DeleteCustomModelOutput{}
	}
	return &DeleteCustomModelOutput{
		Status: string(out.Status),
		Error:  out.Error,
	}
}

func timestampToTime(ts *godo.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.Time
	return &t
}

func jsonNumberFromString(s string) json.Number {
	if s == "" {
		return ""
	}
	return json.Number(s)
}

func jsonNumberFromInt(i int) json.Number {
	return json.Number(strconv.Itoa(i))
}

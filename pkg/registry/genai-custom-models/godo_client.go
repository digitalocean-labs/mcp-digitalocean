package genaicustommodels

import (
	"context"
	"fmt"
	"net/http"

	"github.com/digitalocean/godo"
)

func listCustomModels(ctx context.Context, client *godo.Client, opt *godo.CustomModelListOptions) ([]*CustomModel, *godo.Meta, error) {
	out, resp, err := client.GradientAI.ListCustomModels(ctx, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list custom models: %w", err)
	}
	if resp != nil && resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("failed to list custom models: status %d", resp.StatusCode)
	}
	var meta *godo.Meta
	if out != nil && out.Meta != nil {
		meta = out.Meta
	} else if resp != nil && resp.Meta != nil {
		meta = resp.Meta
	}
	var models []*godo.CustomModel
	if out != nil {
		models = out.Models
	}
	return customModelsFromGodo(models), meta, nil
}

// listAllCustomModels fetches every custom model page from the API.
func listAllCustomModels(ctx context.Context, client *godo.Client) ([]*CustomModel, error) {
	const perPage = 100
	var all []*CustomModel
	page := 1

	for {
		models, meta, err := listCustomModels(ctx, client, &godo.CustomModelListOptions{
			ListOptions: godo.ListOptions{Page: page, PerPage: perPage},
		})
		if err != nil {
			return nil, err
		}

		all = append(all, models...)

		if len(models) == 0 {
			break
		}
		if meta != nil && meta.Pages > 0 && page >= meta.Pages {
			break
		}
		if len(models) < perPage {
			break
		}
		page++
	}

	return all, nil
}

func getCustomModelByUUID(ctx context.Context, client *godo.Client, uuid string) (*CustomModel, error) {
	model, resp, err := client.GradientAI.GetCustomModel(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom model: %w", err)
	}
	if resp != nil && resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get custom model: status %d", resp.StatusCode)
	}
	if model == nil {
		return nil, fmt.Errorf("custom model %q not found", uuid)
	}
	return customModelFromGodo(model), nil
}

func deleteCustomModelByUUID(ctx context.Context, client *godo.Client, uuid string) (*DeleteCustomModelOutput, *godo.Response, error) {
	out, resp, err := client.GradientAI.DeleteCustomModel(ctx, uuid)
	if err != nil {
		return nil, resp, err
	}
	if resp != nil && resp.StatusCode >= 400 {
		return nil, resp, fmt.Errorf("failed to delete custom model: status %d", resp.StatusCode)
	}
	return deleteResponseFromGodo(out), resp, nil
}

func isNotFoundResponse(resp *godo.Response) bool {
	return resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound
}

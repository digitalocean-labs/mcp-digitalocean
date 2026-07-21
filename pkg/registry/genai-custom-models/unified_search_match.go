package genaicustommodels

import (
	"sort"
	"strings"
)

// customModelMatchScore returns a relevance score for a partial query (higher is closer).
// Returns 0 when the model does not match the query.
func customModelMatchScore(cm *CustomModel, queryLower string) int {
	if queryLower == "" {
		return 1
	}

	score := 0
	nameLower := strings.ToLower(cm.Name)

	if nameLower == queryLower {
		score += 1000
	}
	if strings.HasPrefix(nameLower, queryLower) {
		score += 500
	}
	if strings.Contains(nameLower, queryLower) {
		score += 200
	}

	descLower := strings.ToLower(cm.Description)
	if strings.Contains(descLower, queryLower) {
		score += 100
	}

	archLower := strings.ToLower(cm.Architecture)
	if strings.Contains(archLower, queryLower) {
		score += 80
	}

	if cm.Tags != nil {
		for _, tag := range cm.Tags.Tags {
			tagLower := strings.ToLower(tag)
			if tagLower == queryLower {
				score += 150
			} else if strings.Contains(tagLower, queryLower) {
				score += 60
			}
		}
	}

	uuidLower := strings.ToLower(cm.UUID)
	if strings.Contains(uuidLower, queryLower) {
		score += 50
	}

	return score
}

func customModelMatchesQuery(cm *CustomModel, queryLower string) bool {
	return customModelMatchScore(cm, queryLower) > 0
}

type scoredCustomModel struct {
	model *CustomModel
	score int
}

// filterAndRankCustomModels returns custom models nearest to the query, best match first.
// An empty query returns all models sorted by name.
func filterAndRankCustomModels(models []*CustomModel, query string) []*CustomModel {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" {
		out := append([]*CustomModel(nil), models...)
		sort.Slice(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
		return out
	}

	scored := make([]scoredCustomModel, 0, len(models))
	for _, cm := range models {
		if score := customModelMatchScore(cm, queryLower); score > 0 {
			scored = append(scored, scoredCustomModel{model: cm, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return strings.ToLower(scored[i].model.Name) < strings.ToLower(scored[j].model.Name)
	})

	out := make([]*CustomModel, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.model)
	}
	return out
}

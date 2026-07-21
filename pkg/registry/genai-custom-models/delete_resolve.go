package genaicustommodels

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var customModelUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	deleteModelIdentifierRequiredMsg = "either uuid or name is required: provide the exact custom model UUID or exact name the user typed or confirmed. Do not substitute values from a partial-match list."

	deleteModelNameRequiredMsg = "name must be a non-empty string supplied by the user; do not use numbers or other types."

	deleteModelUUIDNameMismatchMsg = "uuid and name refer to different models; provide the exact uuid and exact name for the same model, or pass only one identifier."

	deleteModelNameTypeMsg = "name must be a string when provided."

	deleteConsentRequiredMsg = "confirm_deletion must be true. Before deleting, identify the model (uuid and name), explain that deletion is permanent, and obtain explicit consent (yes) in this conversation. " +
		"Consent is required for every delete. Only pass confirm_deletion: true after the user says yes."

	genaiCustomModelsDeleteConfirmDescription = "Must be true only after the end user has explicitly confirmed deletion in conversation (yes/no in chat). " +
		"Omitted or false is rejected. Not required when the tool only returns a name-match list (partial name with no exact match)."

	genaiCustomModelsDeleteToolDescription = "Delete a custom model by exact UUID or exact name.\n\n" +
		"EXACT IDENTIFIER REQUIRED: Provide uuid OR name (at least one). Only a full UUID (8-4-4-4-12 hex) or an exact model name deletes. " +
		"Partial uuid or partial name returns candidates only — never deletes, even if only one match. " +
		"Never substitute a name or uuid from a prior partial-match list.\n\n" +
		"CONSENT REQUIRED (every delete): Do not call with confirm_deletion: true until the user has explicitly agreed to delete that model. " +
		"Present name, uuid, and that deletion is permanent; ask for yes/no.\n\n" +
		"If both uuid and name are set, they must refer to the same model."
)

// DeleteModelMatchCandidate is a custom model returned when delete-by-name needs disambiguation.
type DeleteModelMatchCandidate struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// DeleteModelUnresolvedOutput is returned when the identifier is not an exact match.
type DeleteModelUnresolvedOutput struct {
	Message                 string                      `json:"message"`
	Query                   string                      `json:"query"`
	QueryField              string                      `json:"query_field"`
	Matches                 []DeleteModelMatchCandidate `json:"matches"`
	RequiresExactMatch      bool                        `json:"requires_exact_match"`
	DoNotSubstituteFromList bool                        `json:"do_not_substitute_name_or_uuid_from_matches"`
}

type deleteTarget struct {
	UUID string
	Name string
}

// resolveCustomModelUUIDByName finds a single model by exact name, or partial name matches.
func resolveCustomModelUUIDByName(name string, models []*CustomModel) (uuid string, unresolved *DeleteModelUnresolvedOutput, err error) {
	active := filterNonDeletedCustomModels(models)

	var exact []*CustomModel
	for _, m := range active {
		if m.Name == name {
			exact = append(exact, m)
		}
	}

	switch len(exact) {
	case 1:
		return exact[0].UUID, nil, nil
	case 0:
		queryLower := strings.ToLower(name)
		var partial []*CustomModel
		for _, m := range active {
			if customModelNameMatchesQuery(m, queryLower) {
				partial = append(partial, m)
			}
		}
		return "", buildDeleteUnresolvedOutput(name, "name", partial), nil
	default:
		return "", nil, fmt.Errorf("multiple custom models named %q; provide uuid to delete one", name)
	}
}

// resolveCustomModelUUIDByUUID finds a model by exact full UUID, or returns partial uuid matches (never auto-deletes on partial).
func resolveCustomModelUUIDByUUID(uuid string, models []*CustomModel) (resolved string, unresolved *DeleteModelUnresolvedOutput, err error) {
	active := filterNonDeletedCustomModels(models)

	if isExactCustomModelUUID(uuid) {
		var exact []*CustomModel
		for _, m := range active {
			if strings.EqualFold(m.UUID, uuid) {
				exact = append(exact, m)
			}
		}
		switch len(exact) {
		case 1:
			return exact[0].UUID, nil, nil
		case 0:
			return "", buildDeleteUnresolvedOutput(uuid, "uuid", nil), nil
		default:
			return "", nil, fmt.Errorf("multiple custom models with uuid %q", uuid)
		}
	}

	queryLower := strings.ToLower(uuid)
	var partial []*CustomModel
	for _, m := range active {
		if strings.Contains(strings.ToLower(m.UUID), queryLower) {
			partial = append(partial, m)
		}
	}
	return "", buildDeleteUnresolvedOutput(uuid, "uuid", partial), nil
}

func isExactCustomModelUUID(uuid string) bool {
	return customModelUUIDPattern.MatchString(uuid)
}

// resolveDeleteTarget resolves an exact model from uuid and/or name. Partial identifiers never return a target.
func resolveDeleteTarget(uuid, name string, models []*CustomModel) (*deleteTarget, *DeleteModelUnresolvedOutput, error) {
	switch {
	case uuid != "" && name != "":
		nameUUID, nameUnresolved, err := resolveCustomModelUUIDByName(name, models)
		if err != nil {
			return nil, nil, err
		}
		if nameUnresolved != nil {
			return nil, nameUnresolved, nil
		}
		uuidResolved, uuidUnresolved, err := resolveCustomModelUUIDByUUID(uuid, models)
		if err != nil {
			return nil, nil, err
		}
		if uuidUnresolved != nil {
			return nil, uuidUnresolved, nil
		}
		if nameUUID != uuidResolved {
			return nil, nil, fmt.Errorf("%s", deleteModelUUIDNameMismatchMsg)
		}
		return &deleteTarget{UUID: nameUUID, Name: name}, nil, nil

	case uuid != "":
		uuidResolved, uuidUnresolved, err := resolveCustomModelUUIDByUUID(uuid, models)
		if err != nil {
			return nil, nil, err
		}
		if uuidUnresolved != nil {
			return nil, uuidUnresolved, nil
		}
		for _, m := range filterNonDeletedCustomModels(models) {
			if strings.EqualFold(m.UUID, uuidResolved) {
				return &deleteTarget{UUID: uuidResolved, Name: m.Name}, nil, nil
			}
		}
		return nil, nil, fmt.Errorf("custom model %q not found", uuidResolved)

	default:
		nameUUID, nameUnresolved, err := resolveCustomModelUUIDByName(name, models)
		if err != nil {
			return nil, nil, err
		}
		if nameUnresolved != nil {
			return nil, nameUnresolved, nil
		}
		return &deleteTarget{UUID: nameUUID, Name: name}, nil, nil
	}
}

func filterNonDeletedCustomModels(models []*CustomModel) []*CustomModel {
	out := make([]*CustomModel, 0, len(models))
	for _, m := range models {
		if m == nil || m.Status == CustomModelStatusDeleted {
			continue
		}
		out = append(out, m)
	}
	return out
}

func customModelNameMatchesQuery(cm *CustomModel, queryLower string) bool {
	return strings.Contains(strings.ToLower(cm.Name), queryLower)
}

func buildDeleteUnresolvedOutput(query, queryField string, matches []*CustomModel) *DeleteModelUnresolvedOutput {
	candidates := make([]DeleteModelMatchCandidate, 0, len(matches))
	for _, m := range matches {
		candidates = append(candidates, DeleteModelMatchCandidate{
			UUID:   m.UUID,
			Name:   m.Name,
			Status: string(m.Status),
		})
	}

	var msg string
	switch queryField {
	case "uuid":
		msg = fmt.Sprintf("no custom model with exact uuid %q; ask the user to provide the full uuid from the matches below — do not delete from a partial uuid", query)
		if len(candidates) == 0 {
			msg = fmt.Sprintf("no custom model with exact uuid %q and no models whose uuids contain that string", query)
		}
		if len(candidates) == 1 {
			msg = fmt.Sprintf("partial uuid %q — one model matched (%s / %q) but deletion requires the exact full uuid; ask the user to confirm the complete uuid", query, candidates[0].UUID, candidates[0].Name)
		}
	default:
		msg = fmt.Sprintf("no custom model with exact name %q; ask the user to provide the exact name (character-for-character) from the matches below — do not substitute a similar name or pass uuid from this list", query)
		if len(candidates) == 0 {
			msg = fmt.Sprintf("no custom model with exact name %q and no models whose names contain that string", query)
		}
		if len(candidates) == 1 {
			msg = fmt.Sprintf("no custom model with exact name %q; one similar model exists (%q) — ask the user to confirm that exact name before deleting; do not delete until they provide it verbatim", query, candidates[0].Name)
		}
	}

	return &DeleteModelUnresolvedOutput{
		Message:                 msg,
		Query:                   query,
		QueryField:              queryField,
		Matches:                 candidates,
		RequiresExactMatch:      true,
		DoNotSubstituteFromList: true,
	}
}

func marshalDeleteUnresolvedResult(out *DeleteModelUnresolvedOutput) (string, error) {
	jsonData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}
	return string(jsonData), nil
}

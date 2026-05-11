package openapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
)

const validationErrorHint = "Use openapi-get-operation for exact parameter names, locations, and body shape."

// humanizeRequestValidationError turns kin-openapi validation errors into short, actionable lines for the model.
func humanizeRequestValidationError(err error) string {
	if err == nil {
		return ""
	}

	var reqErr *openapi3filter.RequestError
	if errors.As(err, &reqErr) {
		reason := strings.TrimSpace(reqErr.Reason)
		if reqErr.Parameter != nil {
			p := reqErr.Parameter
			loc := string(p.In)
			if reason != "" {
				return fmt.Sprintf("Parameter %q (in: %s): %s. %s", p.Name, loc, reason, validationErrorHint)
			}
			return fmt.Sprintf("Parameter %q (in: %s) does not match the OpenAPI spec. %s", p.Name, loc, validationErrorHint)
		}
		if reqErr.RequestBody != nil {
			if reason != "" {
				return fmt.Sprintf("Request body: %s. %s", reason, validationErrorHint)
			}
			return fmt.Sprintf("Request body does not match the OpenAPI schema. %s", validationErrorHint)
		}
		if reason != "" {
			return fmt.Sprintf("%s. %s", reason, validationErrorHint)
		}
	}

	var secErr *openapi3filter.SecurityRequirementsError
	if errors.As(err, &secErr) {
		return fmt.Sprintf("Security requirements not satisfied: %v. %s", err, validationErrorHint)
	}

	return fmt.Sprintf("%v. %s", err, validationErrorHint)
}

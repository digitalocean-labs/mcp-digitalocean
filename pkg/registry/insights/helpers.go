package insights

import "github.com/digitalocean/godo"

func stringArg(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return s
}

func boolArg(args map[string]any, key string) bool {
	b, _ := args[key].(bool)
	return b
}

func intArg(args map[string]any, key string, def int) int {
	if v, ok := args[key].(float64); ok && int(v) > 0 {
		return int(v)
	}
	return def
}

func floatPtrArg(args map[string]any, key string) *float64 {
	if v, ok := args[key].(float64); ok {
		return &v
	}
	return nil
}

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func notificationBindings(args map[string]any) *[]godo.NotificationChannelBinding {
	raw, ok := args["NotificationChannels"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]godo.NotificationChannelBinding, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, godo.NotificationChannelBinding{
			NotificationChannelID: stringArg(m, "NotificationChannelID"),
			NotifyOn:              stringSliceArg(m, "NotifyOn"),
		})
	}
	return &out
}

func queryFilters(args map[string]any) []godo.AlertQueryFilter {
	raw, ok := args["Filters"]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]godo.AlertQueryFilter, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, godo.AlertQueryFilter{
			Field:    stringArg(m, "Field"),
			Operator: stringArg(m, "Operator"),
			Value:    stringArg(m, "Value"),
		})
	}
	return out
}

func alertRuleRequestFromArgs(args map[string]any) *godo.AlertRuleRequest {
	req := &godo.AlertRuleRequest{
		Status: stringArg(args, "Status"),
		Spec: godo.AlertRuleSpec{
			Name: stringArg(args, "Name"),
			Query: godo.AlertMetricsQuery{
				Metric:       stringArg(args, "Metric"),
				Filters:      queryFilters(args),
				ResourceURNs: stringSliceArg(args, "ResourceURNs"),
				Tags:         stringSliceArg(args, "Tags"),
			},
			Thresholds: godo.AlertThresholds{
				Warning:  floatPtrArg(args, "Warning"),
				Critical: floatPtrArg(args, "Critical"),
				Operator: stringArg(args, "Operator"),
			},
			NotificationChannels: notificationBindings(args),
			ReAlertDuration:      stringArg(args, "ReAlertDuration"),
		},
	}
	if window := stringArg(args, "Window"); window != "" {
		req.Spec.Condition = &godo.AlertCondition{Window: window}
	}
	return req
}

func notificationChannelRequestFromArgs(args map[string]any) *godo.NotificationChannelRequest {
	req := &godo.NotificationChannelRequest{Name: stringArg(args, "Name")}
	if to := stringArg(args, "EmailTo"); to != "" {
		req.Email = &godo.EmailNotificationConfig{To: to}
	}
	if ch := stringArg(args, "SlackChannel"); ch != "" {
		req.Slack = &godo.SlackNotificationConfig{
			Channel:    ch,
			WebhookURL: stringArg(args, "SlackWebhookURL"),
		}
	}
	if url := stringArg(args, "WebhookURL"); url != "" {
		wh := &godo.WebhookNotificationConfig{URL: url}
		if user := stringArg(args, "WebhookUsername"); user != "" {
			wh.BasicAuth = &godo.WebhookBasicAuth{Username: user, Password: stringArg(args, "WebhookPassword")}
		}
		if token := stringArg(args, "WebhookBearerToken"); token != "" {
			wh.BearerToken = &godo.WebhookBearerToken{Token: token}
		}
		if secret := stringArg(args, "WebhookSignatureSecret"); secret != "" {
			wh.Signature = &godo.WebhookSignatureConfig{Secret: secret}
		}
		req.Webhook = wh
	}
	return req
}

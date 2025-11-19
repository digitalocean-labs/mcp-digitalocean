# DigitalOcean Insights Tools

This directory provides tool-based handlers for interacting with DigitalOcean insights-related features via the MCP
Server. All insight operations are exposed as tools that accept structured arguments—no resource URIs are used.
Pagination and filtering are supported where applicable.

## Supported Tools

### UptimeCheck

- **uptimecheck-get**
    - Get a specific uptimecheck by its ID.
    - Arguments:
        - `ID` (string, required): The uptimecheck ID.

- **uptimecheck-get-state**
    - Get a specific uptimecheck state by its ID.
    - Arguments:
        - `ID` (string, required): The uptimecheck ID.

- **uptimecheck-list**
    - List uptimechecks with pagination.
    - Arguments:
        - `Page` (number, default: 1): Page number.
        - `PerPage` (number, default: 30): Items per page.

- **uptimecheck-create**
    - Create a new uptimecheck.
    - Arguments:
        - `Name` (string, required): A human-friendly display name.
        - `Type` (string, required): ping, http, or https. The type of health check to perform.
        - `Target` (string, required): The endpoint to perform healthchecks on.
        - `Regions` (array of strings, required): Selected regions to perform healthchecks from. Values: "us_east", "us_west", "eu_west", "se_asia"
        - `Enabled` (bool, required): Whether the check is enabled/disabled.

- **uptimecheck-delete**
    - Delete a specific uptimecheck by its ID.
    - Arguments:
        - `ID` (string, required): The uptimecheck ID.

- **uptimecheck-update**
    - Update an existing uptimecheck by its ID.
    - Arguments:
        - `ID` (string, required): The uptimecheck ID.
        - `Name` (string): A human-friendly display name.
        - `Type` (string): ping, http, or https. The type of health check to perform.
        - `Target` (string): The endpoint to perform healthchecks on.
        - `Regions` (array of strings): Selected regions to perform healthchecks from. Values: "us_east", "us_west", "eu_west", "se_asia"
        - `Enabled` (bool): Whether the check is enabled/disabled.

### UptimeAlert

- **uptimecheck-alert-get**
    - Get uptime check alert information for the check id and alert id.
    - Arguments:
        - `CheckID` (string, required): The uptimecheck ID.
        - `AlertID` (string, required): The uptimecheck alert ID.
        -
- **uptimecheck-alert-list**
    - Get uptime check alert list for the check id.
    - Arguments:
        - `CheckID` (string, required): The uptimecheck ID.

- **uptimecheck-alert-create**
    - Create a new uptimecheck alert.
    - Arguments:
        - `CheckID` (string, required): The uptimecheck ID.
        - `Name` (string): A human-friendly display name.
        - `Type` (string) : The type of alert. values : "latency" "down" "down_global" "ssl_expiry"
        - `Threshold` (number): The threshold value for the alert. This is the value that will trigger the alert.
        - `Comparison` (string): The comparison operator used against the alert's threshold.
        - `Period` (string, required): Period of time the threshold must be exceeded to trigger the alert. values : "
          2m" "3m" "5m" "10m" "15m" "30m" "1h"
        - `Emails` (array of strings, required): Email addresses to notify when the alert is triggered.
        - `Slack` (array of objects, required): Slack notification configuration.
            - Each object should contain:
                - `Channel` (string, required): The Slack channel to post the alert.
                - `URL` (string, required): The Slack webhook URL for posting alerts.

- **uptimecheck-alert-update**
    - Create a new uptimecheck alert.
    - Arguments:
        - `CheckID` (string, required): The uptimecheck ID.
        - `AlertID` (string, required): The uptimecheck alert ID.
        - `Name` (string): A human-friendly display name.
        - `Type` (string) : The type of alert. values : "latency" "down" "down_global" "ssl_expiry"
        - `Threshold` (number): The threshold value for the alert. This is the value that will trigger the alert.
        - `Comparison` (string): The comparison operator used against the alert's threshold.
        - `Period` (string, required): Period of time the threshold must be exceeded to trigger the alert. values : "
          2m" "3m" "5m" "10m" "15m" "30m" "1h"
        - `Emails` (array of strings, required): Email addresses to notify when the alert is triggered.
        - `Slack` (array of objects, required): Slack notification configuration.
            - Each object should contain:
                - `Channel` (string, required): The Slack channel to post the alert.
                - `URL` (string, required): The Slack webhook URL for posting alerts.

### Alert Policy

- **alert-policy-get**
    - Get Alert Policy information by UUID.
    - Arguments:
        - `UUID` (string, required): UUID of the Alert Policy to retrieve.

- **alert-policy-list**
    - List all Alert Policies in your account with pagination.
    - Arguments:
        - `Page` (number, default: 1): Page number for pagination.
        - `PerPage` (number, default: 20): Number of items per page.

- **alert-policy-create**
    - Create a new Alert Policy.
    - Arguments:
        - `Type` (string, required): Type of the Alert Policy (e.g., 'v1/insights/droplet/cpu').
        - `Description` (string, required): Human-readable description of the alert policy.
        - `Compare` (string, required): Comparison operator ('GreaterThan' or 'LessThan').
        - `Value` (number, required): Threshold value for the alert.
        - `Window` (string, required): Time window for the alert ('5m', '10m', '30m', '1h').
        - `Entities` (array of strings): List of resource IDs to monitor.
        - `Tags` (array of strings): List of tags to monitor.
        - `Alerts` (object): Notification settings containing:
            - `Email` (array of strings): List of email addresses.
            - `Slack` (array of objects): List of Slack configurations with:
                - `Channel` (string): Slack channel.
                - `URL` (string): Slack webhook URL.
        - `Enabled` (boolean): Whether the alert policy is enabled.

- **alert-policy-update**
    - Update an existing Alert Policy.
    - Arguments:
        - Same as create, plus:
        - `UUID` (string, required): UUID of the Alert Policy to update.

- **alert-policy-delete**
    - Delete an Alert Policy permanently.
    - Arguments:
        - `UUID` (string, required): UUID of the Alert Policy to delete.

---

## Example Usage

- Get details for check ID 4de7ac8b-495b-4884-9a69-1050c6793cd6:
    - Tool: `uptimecheck-get`
    - Arguments: `{ "ID": "4de7ac8b-495b-4884-9a69-1050c6793cd6" }`

- Get state for check ID 4de7ac8b-495b-4884-9a69-1050c6793cd6:
    - Tool: `uptimecheck-get-state`
    - Arguments: `{ "ID": "4de7ac8b-495b-4884-9a69-1050c6793cd6" }`

- List uptimechecks (page 2, 50 per page):
    - Tool: `uptimecheck-list`
    - Arguments: `{ "Page": 2, "PerPage": 50 }`

- Create a new uptime check:
    - Tool: `uptimecheck-create`
    - Arguments:
      `{ "Name": "Landing page check", "type": "https", "target": "https://www.landingpage.com", "regions": ["us_east","eu_west"], "enabled": true}`

- Update a existing uptime check:
    - Tool: `uptimecheck-update`
    - Arguments:
      `{"ID": "4de7ac8b-495b-4884-9a69-1050c6793cd6"  "Name": "Landing page check", "type": "https", "target": "https://www.landingpage.com", "regions": ["us_east","eu_west"], "enabled": true}`

- Delete uptimecheck with ID 4de7ac8b-495b-4884-9a69-1050c6793cd6:
    - Tool: `uptimecheck-delete`
    - Arguments: `{ "ID": "4de7ac8b-495b-4884-9a69-1050c6793cd6" }`


- Get details for uptimecheck Alert by CheckId 4de7ac8b-495b-4884-9a69-1050c6793ci8 and Alert ID
  4de7ac8b-495b-4884-9a69-1050c6793cd6:
    - Tool: `uptimecheck-alert-get`
    - Arguments:
      `{ "CheckID":"4de7ac8b-495b-4884-9a69-1050c6793ci8" "AlertID": "4de7ac8b-495b-4884-9a69-1050c6793cd6" }`

- Create a new uptimecheck alert:
    - Tool: `uptimecheck-alert-create`
    - Arguments:
      `{ "CheckID":"4de7ac8b-495b-4884-9a69-1050c6793ci8" "name": "Landing page degraded performance" "type": "latency" "threshold": 300 "comparison": "greater_than" "email": ["bob@example.com"] "slack": [{"channel": "Production Alerts","url": "https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ" }] "period": "2m"}`

- Update a existing uptimecheck alert:
    - Tool: `uptimecheck-alert-update`
    - Arguments:
      `{ "CheckID":"4de7ac8b-495b-4884-9a69-1050c6793ci8"  "AlertID": "4de7ac8b-495b-4884-9a69-1050c6793cd6"  "name": "Landing page degraded performance" "type": "latency" "threshold": 300 "comparison": "greater_than"  "email": ["bob@example.com"] "slack": [{"channel": "Production Alerts","url": "https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ" }] "period": "2m"}`

- Delete uptimecheck Alert by CheckId 4de7ac8b-495b-4884-9a69-1050c6793ci8 and AlertID
  4de7ac8b-495b-4884-9a69-1050c6793cd6:
    - Tool: `uptimecheck-alert-delete`
    - Arguments:
      `{ "CheckID":"4de7ac8b-495b-4884-9a69-1050c6793ci8" "AlertID": "4de7ac8b-495b-4884-9a69-1050c6793cd6" }`

- List uptimechecks alerts (page 2, 50 per page):
    - Tool: `uptimechecks-alert-list`
    - Arguments: `{ "CheckID": "4de7ac8b-495b-4884-9a69-1050c6793cd6", "Page": 2, "PerPage": 50 }`

- Get details for Alert Policy with UUID 2dacd69e-44f3-409d-ab58-70df9cf64b92:
    - Tool: `alert-policy-get`
    - Arguments: `{ "UUID": "2dacd69e-44f3-409d-ab58-70df9cf64b92" }`

- List Alert Policies (page 2, 50 per page):
    - Tool: `alert-policy-list`
    - Arguments: `{ "Page": 2, "PerPage": 50 }`

- Create a new Alert Policy for CPU monitoring:
    - Tool: `alert-policy-create`
    - Arguments:
      ```json
      {
        "Type": "v1/insights/droplet/cpu",
        "Description": "Alert when CPU usage is high",
        "Compare": "GreaterThan",
        "Value": 80,
        "Window": "5m",
        "Entities": ["508599038", "509144791"],
        "Tags": ["production"],
        "Alerts": {
          "Email": ["ops@example.com"],
          "Slack": [
            {
              "Channel": "#alerts",
              "URL": "https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ"
            }
          ]
        },
        "Enabled": true
      }
      ```

- Update an existing Alert Policy:
    - Tool: `alert-policy-update`
    - Arguments:
      ```json
      {
        "UUID": "2dacd69e-44f3-409d-ab58-70df9cf64b92",
        "Type": "v1/insights/droplet/cpu",
        "Description": "Alert when CPU usage is very high",
        "Compare": "GreaterThan",
        "Value": 90,
        "Window": "5m",
        "Entities": ["508599038", "509144791", "509144792"],
        "Tags": ["production"],
        "Alerts": {
          "Email": ["ops@example.com", "admin@example.com"],
          "Slack": [
            {
              "Channel": "#alerts",
              "URL": "https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ"
            }
          ]
        },
        "Enabled": true
      }
      ```

- Delete Alert Policy with UUID 2dacd69e-44f3-409d-ab58-70df9cf64b92:
    - Tool: `alert-policy-delete`
    - Arguments: `{ "UUID": "2dacd69e-44f3-409d-ab58-70df9cf64b92" }`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- All responses are returned as JSON-formatted text.
- Error handling is consistent: errors are returned in the tool result with an error flag and message.
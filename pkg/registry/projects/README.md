## DigitalOcean Project Tools

This directory provides tools for managing DigitalOcean Projects via the MCP Server. Projects organize resources (Droplets, Spaces, load balancers, and more) into groups that match how you work. All operations use structured arguments—no resource URIs are required.

These tools are registered when either the `droplets` or `accounts` service is enabled.

## Supported Tools

### Projects

- **project-list**
  - List projects with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

- **project-get**
  - Get a project by ID.
  - Arguments:
    - `ID` (string, required): Project UUID, or `"default"` for the account default project.

- **project-get-default**
  - Get the account's default project. This is the same API call as `project-get` with `ID` `"default"`; both read `/v2/projects/default`.
  - Arguments: _none_

- **project-create**
  - Create a new project.
  - Arguments:
    - `Name` (string, required): Name of the project.
    - `Purpose` (string, optional): Purpose of the project. Preferred values include `"Web Application"`, `"Service or API"`, `"Website or blog"`, `"Mobile Application"`, `"Machine learning / AI / Data processing"`, `"IoT"`, `"Operational / Developer tooling"`, `"Just trying out DigitalOcean"`, and `"Class project / Educational purposes"`. Other values are stored as `"Other: <value>"`.
    - `Description` (string, optional): Description of the project.
    - `Environment` (string, optional): One of `"Development"`, `"Staging"`, or `"Production"`.

---

## Example Usage

- List projects (first page):
  - Tool: `project-list`
  - Arguments: `{ "Page": 1, "PerPage": 30 }`

- Get the default project:
  - Tool: `project-get-default`
  - Arguments: `{}`

- Create a project:
  - Tool: `project-create`
  - Arguments: `{ "Name": "my-web-api", "Purpose": "Service or API", "Environment": "Production" }`

---

## Notes

- Use `project-list` / `project-create` together with `droplet-create`'s optional `ProjectID` argument to place new Droplets into a project. `ProjectID` may be a project UUID or `"default"`.
- Pagination defaults to page 1 with 30 items per page when omitted.

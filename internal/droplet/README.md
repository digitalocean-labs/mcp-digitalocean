# Droplet MCP Tools

This directory contains tools for managing DigitalOcean Droplets, Images, and Sizes via the MCP Server. All operations are exposed as tools with argument-based input—no resource URIs are used. Pagination and filtering are supported where applicable.

---

## Supported Tools

### Droplet Tools

- **digitalocean-droplet-create**  
  Create a new Droplet.  
  **Arguments:**  
  - `Name` (string, required): Name of the Droplet  
  - `Size` (string, required): Slug of the Droplet size (e.g., `s-1vcpu-1gb`)  
  - `ImageID` (number, required): ID of the image to use  
  - `Region` (string, required): Slug of the region (e.g., `nyc3`)  
  - `Backup` (boolean, optional, default: false): Enable backups  
  - `Monitoring` (boolean, optional, default: false): Enable monitoring

- **digitalocean-droplet-delete**  
  Delete a Droplet.  
  **Arguments:**  
  - `ID` (number, required): ID of the Droplet to delete

- **digitalocean-droplet-get**  
  Get information about a specific Droplet by its ID.  
  **Arguments:**  
  - `ID` (number, required): Droplet ID

- **digitalocean-droplet-action-get**  
  Get information about a specific action performed on a Droplet.  
  **Arguments:**  
  - `DropletID` (number, required): Droplet ID  
  - `ActionID` (number, required): Action ID

- **digitalocean-droplet-power-cycle**  
- **digitalocean-droplet-power-on**  
- **digitalocean-droplet-power-off**  
- **digitalocean-droplet-shutdown**  
- **digitalocean-droplet-reboot**  
  Power and state management for a Droplet.  
  **Arguments:**  
  - `ID` (number, required): Droplet ID

- **digitalocean-droplet-restore**  
  Restore a Droplet from a backup/snapshot.  
  - `ID` (number, required): Droplet ID  
  - `ImageID` (number, required): Backup/snapshot image ID

- **digitalocean-droplet-resize**  
  Resize a Droplet.  
  - `ID` (number, required): Droplet ID  
  - `Size` (string, required): New size slug  
  - `ResizeDisk` (boolean, optional, default: false): Whether to resize the disk

- **digitalocean-droplet-rebuild**  
  Rebuild a Droplet from an image.  
  - `ID` (number, required): Droplet ID  
  - `ImageID` (number, required): Image ID

- **digitalocean-droplet-rebuild-by-slug**  
  Rebuild a Droplet using an image slug.  
  - `ID` (number, required): Droplet ID  
  - `ImageSlug` (string, required): Image slug

- **digitalocean-droplet-snapshot**  
  Take a snapshot of a Droplet.  
  - `ID` (number, required): Droplet ID  
  - `Name` (string, required): Name for the snapshot

- **digitalocean-droplet-enable-ipv6**  
- **digitalocean-droplet-enable-private-net**  
- **digitalocean-droplet-enable-backups**  
- **digitalocean-droplet-disable-backups**  
  Enable/disable features on a Droplet.  
  - `ID` (number, required): Droplet ID

- **digitalocean-droplet-get-neighbors**  
  Get neighbors of a Droplet.  
  - `ID` (number, required): Droplet ID

- **digitalocean-droplet-get-kernels**  
  Get available kernels for a Droplet.  
  - `ID` (number, required): Droplet ID

- **digitalocean-droplet-change-kernel**  
  Change a Droplet's kernel.  
  - `ID` (number, required): Droplet ID  
  - `KernelID` (number, required): Kernel ID

- **digitalocean-droplet-rename**  
  Rename a Droplet.  
  - `ID` (number, required): Droplet ID  
  - `Name` (string, required): New name

- **digitalocean-droplet-password-reset**  
  Reset password for a Droplet.  
  - `ID` (number, required): Droplet ID

#### Tag-based Bulk Actions

- **digitalocean-droplet-power-cycle-by-tag**  
- **digitalocean-droplet-power-on-by-tag**  
- **digitalocean-droplet-power-off-by-tag**  
- **digitalocean-droplet-shutdown-by-tag**  
- **digitalocean-droplet-enable-backups-by-tag**  
- **digitalocean-droplet-disable-backups-by-tag**  
- **digitalocean-droplet-snapshot-by-tag**  
- **digitalocean-droplet-enable-ipv6-by-tag**  
- **digitalocean-droplet-enable-private-net-by-tag**  
  All require:  
  - `Tag` (string, required): Tag of the droplets  
  Some require:  
  - `Name` (string, required): Name for the snapshot (for snapshot-by-tag)

---

### Image Tools

- **digitalocean-image-list**  
  List all available distribution images. Supports pagination.  
  **Arguments:**  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Items per page

- **digitalocean-image-get**  
  Get a specific image by its numeric ID.  
  **Arguments:**  
  - `ID` (number, required): Image ID

---

### Size Tools

- **digitalocean-size-list**  
  List all available Droplet sizes. Supports pagination.  
  **Arguments:**  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Items per page

---

## Example Usage

- **Create a new Droplet:**  
  Tool: `digitalocean-droplet-create`  
  Arguments:  
    - `Name`: `"web-1"`  
    - `Size`: `"s-1vcpu-1gb"`  
    - `ImageID`: `123456`  
    - `Region`: `"nyc3"`  
    - `Backup`: `true`  
    - `Monitoring`: `true`

- **Get a Droplet by ID:**  
  Tool: `digitalocean-droplet-get`  
  Arguments:  
    - `ID`: `12345`

- **Get a Droplet action:**  
  Tool: `digitalocean-droplet-action-get`  
  Arguments:  
    - `DropletID`: `12345`  
    - `ActionID`: `67890`

- **List all distribution images:**  
  Tool: `digitalocean-image-list`  
  Arguments: `{}`

- **Get image by ID:**  
  Tool: `digitalocean-image-get`  
  Arguments:  
    - `ID`: `7890`

- **List all Droplet sizes:**  
  Tool: `digitalocean-size-list`  
  Arguments: `{}`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- Tag-based tools allow you to perform bulk actions on all Droplets with a given tag.
- All responses are returned in JSON format for easy parsing and integration.
- For endpoints that require an ID or tag, provide the appropriate value in your query.
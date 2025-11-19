# Droplet MCP Tools

This directory contains tools for managing DigitalOcean Droplets, Images, and Sizes via the MCP Server. All operations are exposed as tools with argument-based input—no resource URIs are used. Pagination and filtering are supported where applicable.

---

## Supported Tools

### Droplet Tools

- **droplet-create**  
  Create a new Droplet.  
  **Arguments:**  
  - `Name` (string, required): Name of the Droplet  
  - `Size` (string, required): Slug of the Droplet size (e.g., `s-1vcpu-1gb`)  
  - `ImageID` (number, required): ID of the image to use  
  - `Region` (string, required): Slug of the region (e.g., `nyc3`)  
  - `Backup` (boolean, optional, default: false): Enable backups  
  - `Monitoring` (boolean, optional, default: false): Enable monitoring

- **droplet-delete**  
  Delete a Droplet.  
  **Arguments:**  
  - `ID` (number, required): ID of the Droplet to delete

- **droplet-get**  
  Get information about a specific Droplet by its ID.  
  **Arguments:**  
  - `ID` (number, required): Droplet ID

- **droplet-list**  
  List all droplets for the user. Supports pagination.  
  **Arguments:**  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Items per page

---

### Droplet Actions Tools

- **droplet-action**  
  Get information about a specific action performed on a Droplet.  
  **Arguments:**  
  - `DropletID` (number, required): Droplet ID  
  - `ActionID` (number, required): Action ID

- **droplet-reboot**  
  Reboot a Droplet.  
  **Arguments:**  
  - `ID` (number, required): Droplet ID

---

## Example Usage

- **Create a new Droplet:**  
  Tool: `droplet-create`  
  Arguments:  
    - `Name`: `"web-1"`  
    - `Size`: `"s-1vcpu-1gb"`  
    - `ImageID`: `123456`  
    - `Region`: `"nyc3"`  
    - `Backup`: `true`  
    - `Monitoring`: `true`

- **Get a Droplet by ID:**  
  Tool: `droplet-get`  
  Arguments:  
    - `ID`: `12345`

- **Reboot a Droplet:**  
  Tool: `droplet-reboot`  
  Arguments:  
    - `ID`: `12345`

- **reset-droplet-password**  
  Reset password for a Droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

- **rename-droplet**  
  Rename a Droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `Name` (string, required): New name

- **change-kernel-droplet**  
  Change a Droplet's kernel.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `KernelID` (number, required): Kernel ID

- **enable-ipv6-droplet**
- **enable-private-net-droplet**
- **enable-backups-droplet**
- **disable-backups-droplet**  
  Enable/disable features on a Droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

#### Tag-based Bulk Actions

- **power-cycle-droplets-tag**
- **power-on-droplets-tag**
- **power-off-droplets-tag**
- **shutdown-droplets-tag**
- **enable-backups-droplets-tag**
- **disable-backups-droplets-tag**
- **snapshot-droplets-tag**
- **enable-ipv6-droplets-tag**
- **enable-private-net-droplets-tag**  
  All require:
  - `Tag` (string, required): Tag of the droplets  
    Some require:
  - `Name` (string, required): Name for the snapshot (for snapshot-by-tag)

---

### Additional Droplet Actions Tools

- **rebuild-droplet-by-slug**  
  Rebuild a droplet using an image slug.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `ImageSlug` (string, required): Slug of the image to rebuild from

- **power-cycle-droplet**  
  Power cycle a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

- **power-on-droplet**  
  Power on a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

- **power-off-droplet**  
  Power off a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

- **shutdown-droplet**  
  Shutdown a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID

- **restore-droplet**  
  Restore a droplet from a backup/snapshot.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `ImageID` (number, required): ID of the backup/snapshot image

- **resize-droplet**  
  Resize a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `Size` (string, required): Slug of the new size (e.g., s-1vcpu-1gb)
  - `ResizeDisk` (boolean, optional, default: false): Whether to resize the disk

- **rebuild-droplet**  
  Rebuild a droplet from an image.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `ImageID` (number, required): ID of the image to rebuild from

- **snapshot-droplet**  
  Take a snapshot of a droplet.  
  **Arguments:**
  - `ID` (number, required): Droplet ID
  - `Name` (string, required): Name for the snapshot

---

### Image Tools

- **image-list**  
  List all available distribution images. Supports pagination.  
  **Arguments:**
  - `Page` (number, default: 1): Page number
  - `PerPage` (number, default: 50): Items per page

- **image-get**  
  Get a specific image by its numeric ID.  
  **Arguments:**
  - `ID` (number, required): Image ID

---

### Size Tools

- **size-list**  
  List all available Droplet sizes. Supports pagination.  
  **Arguments:**
  - `Page` (number, default: 1): Page number
  - `PerPage` (number, default: 50): Items per page

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- Tag-based tools allow you to perform bulk actions on all Droplets with a given tag.
- All responses are returned in JSON format for easy parsing and integration.
- For endpoints that require an ID or tag, provide the appropriate value in your query.

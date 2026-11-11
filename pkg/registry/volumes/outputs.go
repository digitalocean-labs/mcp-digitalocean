package volumes

import (
	"time"

	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// volumeSummary is the per-volume shape volume-list has always returned: the
// identifying, sizing and attachment fields, without the rest of the API
// resource. It was an inline map literal; naming it lets one type drive both
// the declared schema and the emitted payload. A map serialises every key it
// holds, so none of these fields are omitempty.
type volumeSummary struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	SizeGigaBytes   int64        `json:"size_gigabytes"`
	Region          *godo.Region `json:"region"`
	Description     string       `json:"description"`
	FilesystemType  string       `json:"filesystem_type"`
	FilesystemLabel string       `json:"filesystem_label"`
	Tags            []string     `json:"tags"`
	CreatedAt       time.Time    `json:"created_at"`
	DropletIDs      []int        `json:"droplet_ids"`
}

// newVolumeSummary projects a volume onto the list summary.
func newVolumeSummary(volume godo.Volume) volumeSummary {
	return volumeSummary{
		ID:              volume.ID,
		Name:            volume.Name,
		SizeGigaBytes:   volume.SizeGigaBytes,
		Region:          volume.Region,
		Description:     volume.Description,
		FilesystemType:  volume.FilesystemType,
		FilesystemLabel: volume.FilesystemLabel,
		Tags:            volume.Tags,
		CreatedAt:       volume.CreatedAt,
		DropletIDs:      volume.DropletIDs,
	}
}

// snapshotSummary is the per-snapshot shape volume-snapshot-list has always
// returned. It was likewise an inline map literal, and likewise keeps every
// field unconditionally: "created_at" carries godo's Created field, which the
// resource itself declares omitempty but the map always wrote.
type snapshotSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ResourceID    string   `json:"resource_id"`
	ResourceType  string   `json:"resource_type"`
	Regions       []string `json:"regions"`
	MinDiskSize   int      `json:"min_disk_size"`
	SizeGigaBytes float64  `json:"size_gigabytes"`
	Tags          []string `json:"tags"`
	CreatedAt     string   `json:"created_at"`
}

// newSnapshotSummary projects a snapshot onto the list summary.
func newSnapshotSummary(snapshot godo.Snapshot) snapshotSummary {
	return snapshotSummary{
		ID:            snapshot.ID,
		Name:          snapshot.Name,
		ResourceID:    snapshot.ResourceID,
		ResourceType:  snapshot.ResourceType,
		Regions:       snapshot.Regions,
		MinDiskSize:   snapshot.MinDiskSize,
		SizeGigaBytes: snapshot.SizeGigaBytes,
		Tags:          snapshot.Tags,
		CreatedAt:     snapshot.Created,
	}
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is either a bare resource or an array, so each one needs
// an envelope to satisfy MCP's object-root requirement. volumeOut is shared by
// the create and get tools, and snapshotOut by the snapshot create and get
// tools, because each pair returns the same resource shape. actionOut is shared
// by attach, detach, resize and action-get: every volume action resolves to the
// same pollable godo.Action, which is the point of returning it.
//
// volume-delete and volume-snapshot-delete stay text-only: each returns a fixed
// success message rather than a resource, so an output schema would describe
// nothing.
var (
	volumeOut       = common.NewOutput[*godo.Volume]("volume")
	volumeListOut   = common.NewOutput[[]volumeSummary]("volumes")
	snapshotOut     = common.NewOutput[*godo.Snapshot]("snapshot")
	snapshotListOut = common.NewOutput[[]snapshotSummary]("snapshots")
	actionOut       = common.NewOutput[*godo.Action]("action")
	actionListOut   = common.NewOutput[[]godo.Action]("actions")
)

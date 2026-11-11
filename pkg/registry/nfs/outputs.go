package nfs

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// fileShareSummary is the per-share shape nfs-file-share-list has always
// returned: the share's identity, sizing and mount details, without the
// access points the full resource carries. It was an inline map literal;
// naming it lets one type drive both the declared schema and the emitted
// payload. A map serialises every key it holds, so none of these fields are
// omitempty.
type fileShareSummary struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	SizeGib         int                 `json:"size_gib"`
	Region          string              `json:"region"`
	PerformanceTier string              `json:"performance_tier"`
	CreatedAt       string              `json:"created_at"`
	VpcIDs          []string            `json:"vpc_ids"`
	Status          godo.NfsShareStatus `json:"status"`
	MountPath       string              `json:"mount_path"`
	Host            string              `json:"host"`
}

// newFileShareSummary projects a share onto the list summary.
func newFileShareSummary(fileShare *godo.Nfs) fileShareSummary {
	return fileShareSummary{
		ID:              fileShare.ID,
		Name:            fileShare.Name,
		SizeGib:         fileShare.SizeGib,
		Region:          fileShare.Region,
		PerformanceTier: fileShare.PerformanceTier,
		CreatedAt:       fileShare.CreatedAt,
		VpcIDs:          fileShare.VpcIDs,
		Status:          fileShare.Status,
		MountPath:       fileShare.MountPath,
		Host:            fileShare.Host,
	}
}

// snapshotSummary is the per-snapshot shape nfs-snapshot-list has always
// returned. It was an inline map literal, and is named here for the same
// reason as fileShareSummary.
type snapshotSummary struct {
	ID        string                 `json:"id"`
	ShareID   string                 `json:"share_id"`
	Name      string                 `json:"name"`
	SizeGib   int                    `json:"size_gib"`
	Region    string                 `json:"region"`
	CreatedAt string                 `json:"created_at"`
	Status    godo.NfsSnapshotStatus `json:"status"`
}

// newSnapshotSummary projects a snapshot onto the list summary.
func newSnapshotSummary(snapshot *godo.NfsSnapshot) snapshotSummary {
	return snapshotSummary{
		ID:        snapshot.ID,
		ShareID:   snapshot.ShareID,
		Name:      snapshot.Name,
		SizeGib:   snapshot.SizeGib,
		Region:    snapshot.Region,
		CreatedAt: snapshot.CreatedAt,
		Status:    snapshot.Status,
	}
}

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is either a bare resource or an array, so each one needs
// an envelope to satisfy MCP's object-root requirement. The field names follow
// the keys the NFS API itself uses ("share"/"shares", "snapshot"/"snapshots").
// shareOut is shared by the create and get tools, which return the same
// resource. actionOut is shared by all six tools in nfs_actions.go: each one
// resolves to the same pollable action object describing the mutation it
// started.
//
// nfs-file-share-delete and nfs-snapshot-delete stay text-only: each returns a
// fixed success message rather than a resource, so an output schema would
// describe nothing.
var (
	shareOut        = common.NewOutput[*godo.Nfs]("share")
	shareListOut    = common.NewOutput[[]fileShareSummary]("shares")
	snapshotOut     = common.NewOutput[*godo.NfsSnapshot]("snapshot")
	snapshotListOut = common.NewOutput[[]snapshotSummary]("snapshots")
	actionOut       = common.NewOutput[*godo.NfsAction]("action")
)

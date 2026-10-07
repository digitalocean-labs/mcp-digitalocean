package signals

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

var (
	consentOut        = common.NewObjectOutput[*godo.SignalsAgentConsent]()
	consentSetOut     = common.NewObjectOutput[*godo.SignalsConsentRecord]()
	consentsListOut   = common.NewObjectOutput[*godo.SignalsListConsentsResponse]()
	sessionsOut       = common.NewObjectOutput[*godo.SignalsListAgentSessionsResponse]()
	segmentsOut       = common.NewObjectOutput[*godo.SignalsListSegmentsResponse]()
	dialoguesOut      = common.NewObjectOutput[*godo.SignalsSessionDialoguesResponse]()
	segmentOut        = common.NewObjectOutput[*godo.SignalsSegmentDetailResponse]()
	reportOut         = common.NewObjectOutput[*godo.SignalsReport]()
	exportsOut        = common.NewObjectOutput[*godo.SignalsListExportsResponse]()
	exportJobOut      = common.NewObjectOutput[*godo.SignalsExportJob]()
	exportDownloadOut = common.NewObjectOutput[*godo.SignalsExportDownload]()
	exportOptionsOut  = common.NewObjectOutput[*godo.SignalsExportOptions]()
)

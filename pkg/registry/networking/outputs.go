package networking

import (
	"github.com/digitalocean/godo"

	"mcp-digitalocean/pkg/registry/common"
)

// Output contracts for this package's tools; see the marketplace package for
// the convention.
//
// Every payload here is either a bare resource or a bare array, so each one
// takes an envelope to satisfy MCP's object-root requirement. Vars are shared
// by every tool that returns the same shape: certificateOut by both create
// tools and get, domainOut by create and get, domainRecordOut by record
// create/get/edit, firewallOut by create and get, loadBalancerOut by create,
// get and update, vpcOut by create and get, vpcPeeringOut by create and get,
// partnerAttachmentOut by create, get and update, and actionOut by
// reserved-ip-assign and -unassign, since both resolve to the same pollable
// godo.Action.
//
// byoipPrefixCreateOut is separate from byoipPrefixOut even though both are
// published under "byoip_prefix": the create endpoint answers with the
// narrower BYOIPPrefixCreateResp, not the full prefix.
//
// reservedIPOut and reservedIPListOut are the one weak spot. reserved-ip-get,
// -list and -reserve each dispatch on a "Type" argument and return either the
// IPv4 or the IPv6 resource, which have different shapes (region vs
// region_slug, locked/project_id vs reserved_at). A tool declares a single
// outputSchema, and the two cannot be merged without changing the emitted
// JSON, so the payload stays any: the envelope key is described, the resource
// inside is left open.
//
// Left text-only, because each returns a fixed confirmation string rather
// than a resource, so an output schema would describe nothing: every delete
// and release tool (certificate-delete, domain-delete, domain-record-delete,
// firewall-delete, lb-delete, lb-delete-cache, vpc-delete,
// vpc-peering-delete, byoip-prefix-delete, partner-attachment-delete,
// reserved-ip-release) and every membership/rule mutation
// (firewall-add-droplets, firewall-remove-droplets, firewall-add-tags,
// firewall-remove-tags, firewall-add-rules, firewall-remove-rules,
// lb-add-droplets, lb-remove-droplets, lb-add-fwd-rules, lb-remove-fwd-rules).
var (
	certificateOut     = common.NewOutput[*godo.Certificate]("certificate")
	certificateListOut = common.NewOutput[[]godo.Certificate]("certificates")

	domainOut           = common.NewOutput[*godo.Domain]("domain")
	domainListOut       = common.NewOutput[[]godo.Domain]("domains")
	domainRecordOut     = common.NewOutput[*godo.DomainRecord]("domain_record")
	domainRecordListOut = common.NewOutput[[]godo.DomainRecord]("domain_records")

	firewallOut     = common.NewOutput[*godo.Firewall]("firewall")
	firewallListOut = common.NewOutput[[]godo.Firewall]("firewalls")

	loadBalancerOut     = common.NewOutput[*godo.LoadBalancer]("load_balancer")
	loadBalancerListOut = common.NewOutput[[]godo.LoadBalancer]("load_balancers")

	vpcOut           = common.NewOutput[*godo.VPC]("vpc")
	vpcListOut       = common.NewOutput[[]*godo.VPC]("vpcs")
	vpcMemberListOut = common.NewOutput[[]*godo.VPCMember]("members")

	vpcPeeringOut     = common.NewOutput[*godo.VPCPeering]("vpc_peering")
	vpcPeeringListOut = common.NewOutput[[]*godo.VPCPeering]("vpc_peerings")

	byoipPrefixOut             = common.NewOutput[*godo.BYOIPPrefix]("byoip_prefix")
	byoipPrefixListOut         = common.NewOutput[[]*godo.BYOIPPrefix]("byoip_prefixes")
	byoipPrefixCreateOut       = common.NewOutput[*godo.BYOIPPrefixCreateResp]("byoip_prefix")
	byoipPrefixResourceListOut = common.NewOutput[[]godo.BYOIPPrefixResource]("resources")

	partnerAttachmentOut     = common.NewOutput[*godo.PartnerAttachment]("partner_attachment")
	partnerAttachmentListOut = common.NewOutput[[]*godo.PartnerAttachment]("partner_attachments")
	serviceKeyOut            = common.NewOutput[*godo.ServiceKey]("service_key")
	bgpAuthKeyOut            = common.NewOutput[*godo.BgpAuthKey]("bgp_auth_key")

	reservedIPOut     = common.NewOutput[any]("reserved_ip")
	reservedIPListOut = common.NewOutput[any]("reserved_ips")
	actionOut         = common.NewOutput[*godo.Action]("action")
)

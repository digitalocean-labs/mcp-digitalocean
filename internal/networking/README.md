# Networking MCP Tools

This directory contains tools and resources for managing DigitalOcean networking features via the MCP Server. These tools enable you to create, modify, and query networking resources such as domains, certificates, firewalls, reserved IPs, VPCs, CDNs, and partner attachments.

---

## Supported Tools

### Domains

- **`digitalocean-domain-create`**
  Create a new domain.
  **Arguments:**
  - `Name` (string, required): Name of the domain
  - `IPAddress` (string, required): IP address for the domain

- **`digitalocean-domain-delete`**
  Delete a domain.
  - `Name` (string, required): Name of the domain to delete

- **`digitalocean-domain-record-create`**
  Create a new domain record.
  - `Domain` (string, required): Domain name
  - `Type` (string, required): Record type (e.g., A, CNAME, TXT)
  - `Name` (string, required): Record name
  - `Data` (string, required): Record data

- **`digitalocean-domain-record-delete`**
  Delete a domain record.
  - `Domain` (string, required): Domain name
  - `RecordID` (number, required): ID of the record to delete

- **`digitalocean-domain-record-edit`**
  Edit a domain record.
  - `Domain` (string, required): Domain name
  - `RecordID` (number, required): ID of the record to edit
  - `Type` (string, required): Record type
  - `Name` (string, required): Record name
  - `Data` (string, required): Record data

- **`digitalocean-domain-get`**  
  Get domain information by name.  
  - `Name` (string, required): Name of the domain

- **`digitalocean-domain-list`**  
  List domains with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

- **`digitalocean-domain-record-get`**  
  Get a domain record by domain name and record ID.  
  - `Domain` (string, required): Domain name  
  - `RecordID` (number, required): ID of the domain record

- **`digitalocean-domain-record-list`**  
  List domain records for a domain with pagination.  
  - `Domain` (string, required): Domain name  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### Certificates

- **`digitalocean-certificate-create`**
  Create a new certificate.
  - `Name` (string, required): Name of the certificate
  - `PrivateKey` (string, required): Private key
  - `LeafCertificate` (string, required): Leaf certificate
  - `CertificateChain` (string, required): Certificate chain

- **`digitalocean-certificate-delete`**
  Delete a certificate.
  - `ID` (string, required): ID of the certificate to delete

- **`digitalocean-certificate-get`**  
  Get certificate information by ID.  
  - `ID` (string, required): ID of the certificate

- **`digitalocean-certificate-list`**  
  List certificates with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### Firewalls

- **`digitalocean-firewall-create`**
  Create a new firewall.
  - `Name` (string, required): Name of the firewall
  - `InboundProtocol` (string, required): Protocol for inbound rule
  - `InboundPortRange` (string, required): Port range for inbound rule
  - `InboundSource` (string, required): Source address for inbound rule
  - `OutboundProtocol` (string, required): Protocol for outbound rule
  - `OutboundPortRange` (string, required): Port range for outbound rule
  - `OutboundDestination` (string, required): Destination address for outbound rule
  - `DropletIDs` (array of numbers, optional): Droplet IDs to apply the firewall to
  - `Tags` (array of strings, optional): Tags to apply the firewall to

- **`digitalocean-firewall-delete`**
  Delete a firewall.
  - `ID` (string, required): ID of the firewall to delete

- **`digitalocean-firewall-add-tags`**
  Add one or more tags to a firewall.
  - `ID` (string, required): ID of the firewall to update tags
  - `Tags` (array of strings, required): Tags to apply the firewall to

- **`digitalocean-firewall-remove-tags`**
  Remove one or more tags from a firewall.
  - `ID` (string, required): ID of the firewall to update tags
  - `Tags` (array of strings, required): Tags to remove from the firewall

- **`digitalocean-firewall-add-droplets`**
  Add one or more droplets to a firewall.
  - `ID` (string, required): ID of the firewall to apply to droplets
  - `DropletIDs` (array of numbers, required): Droplet IDs to apply the firewall to

- **`digitalocean-firewall-remove-droplets`**
  Remove one or more droplets from a firewall.
  - `ID` (string, required): ID of the firewall to remove droplets from
  - `DropletIDs` (array of numbers, required): Droplet IDs to remove from the firewall

- **`digitalocean-firewall-add-rules`**
  Add one or more rules to a firewall.
  - `ID` (string, required): ID of the firewall to add rules to
  - `InboundRules` (array of objects, optional): Inbound rules to add
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Sources` (array of strings, required): Source IP addresses or CIDR blocks
  - `OutboundRules` (array of objects, optional): Outbound rules to add
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Destinations` (array of strings, required): Destination IP addresses or CIDR blocks

- **`digitalocean-firewall-remove-rules`**
  Remove one or more rules from a firewall.
  - `ID` (string, required): ID of the firewall to remove rules from
  - `InboundRules` (array of objects, optional): Inbound rules to remove
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Sources` (array of strings, required): Source IP addresses or CIDR blocks
  - `OutboundRules` (array of objects, optional): Outbound rules to remove
    - `Protocol` (string, required): Protocol (tcp, udp, icmp)
    - `PortRange` (string, required): Port range (e.g., '80', '443', '8000-8080')
    - `Destinations` (array of strings, required): Destination IP addresses or CIDR blocks

- **`digitalocean-firewall-get`**  
  Get firewall information by ID.  
  - `ID` (string, required): ID of the firewall

- **`digitalocean-firewall-list`**  
  List firewalls with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---


### Reserved IPs

- **`digitalocean-reserved-ip-reserve`**
  Reserve a new IPv4 or IPv6.
  - `Region` (string, required): Region to reserve the IP in
  - `Type` (string, required): Type of IP to reserve (`ipv4` or `ipv6`)

- **`digitalocean-reserved-ip-release`**
  Release a reserved IPv4 or IPv6.
  - `IP` (string, required): The reserved IP to release
  - `Type` (string, required): Type of IP to release (`ipv4` or `ipv6`)

- **`digitalocean-reserved-ip-assign`**
  Assign a reserved IP to a droplet.
  - `IP` (string, required): The reserved IP to assign
  - `DropletID` (number, required): The ID of the droplet
  - `Type` (string, required): Type of IP (`ipv4` or `ipv6`)

- **`digitalocean-reserved-ip-unassign`**
  Unassign a reserved IP from a droplet.
  - `IP` (string, required): The reserved IP to unassign
  - `Type` (string, required): Type of IP (`ipv4` or `ipv6`)

- **`digitalocean-reserved-ipv4-list`**
  List reserved IPv4 addresses with pagination.
  - `Page` (number, optional, default: 1): Page number
  - `PerPage` (number, optional, default: 20): Items per page

- **`digitalocean-reserved-ipv6-list`**
  List reserved IPv6 addresses with pagination.
  - `Page` (number, optional, default: 1): Page number
  - `PerPage` (number, optional, default: 20): Items per page

- **`digitalocean-reserved-ipv4-list`**  
  List reserved IPv4 addresses with pagination.  
  - `Page` (number, optional, default: 1): Page number  
  - `PerPage` (number, optional, default: 20): Items per page

- **`digitalocean-reserved-ipv6-list`**  
  List reserved IPv6 addresses with pagination.  
  - `Page` (number, optional, default: 1): Page number  
  - `PerPage` (number, optional, default: 20): Items per page

- **`digitalocean-reserved-ipv4-get`**  
  Get reserved IPv4 information by IP.  
  - `IP` (string, required): The reserved IPv4 address

- **`digitalocean-reserved-ipv6-get`**  
  Get reserved IPv6 information by IP.  
  - `IP` (string, required): The reserved IPv6 address
  
---

### VPC Peering

- **`digitalocean-vpc-peering-create`**
  Create a new VPC Peering connection between two VPCs.
  - `Name` (string, required): Name for the Peering connection
  - `Vpc1` (string, required): ID of the first VPC
  - `Vpc2` (string, required): ID of the second VPC

- **`digitalocean-vpc-peering-delete`**
  Delete a VPC Peering connection.
  - `ID` (string, required): ID of the VPC Peering connection to delete

- **`digitalocean-vpc-peering-get`**  
  Get VPC Peering information by ID.  
  - `ID` (string, required): ID of the VPC Peering connection

- **`digitalocean-vpc-peering-list`**  
  List VPC Peering connections with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### VPCs

- **`digitalocean-vpc-create`**
  Create a new VPC.
  - `Name` (string, required): Name of the VPC
  - `Region` (string, required): Region slug (e.g., nyc3)
  - `Subnet` (string, optional): Optional subnet CIDR block (e.g., 10.10.0.0/20)
  - `Description` (string, optional): Optional description for the VPC

- **`digitalocean-vpc-list-members`**
  List members of a VPC.
  - `ID` (string, required): ID of the VPC

- **`digitalocean-vpc-delete`**
  Delete a VPC.
  - `ID` (string, required): ID of the VPC to delete

- **`digitalocean-vpc-get`**  
  Get VPC information by ID.  
  - `ID` (string, required): ID of the VPC

- **`digitalocean-vpc-list`**  
  List VPCs with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

### Partner Attachments

- **`digitalocean-partner-attachment-create`**
  Create a new partner attachment.
  - `Name` (string, required): Name of the partner attachment
  - `Region` (string, required): Region for the partner attachment
  - `Bandwidth` (number, required): Bandwidth in Mbps

- **`digitalocean-partner-attachment-delete`**
  Delete a partner attachment.
  - `ID` (string, required): ID of the partner attachment

- **`digitalocean-partner-attachment-get-service-key`**
  Get the service key of a partner attachment.
  - `ID` (string, required): ID of the partner attachment

- **`digitalocean-partner-attachment-get-bgp-config`**
  Get the BGP configuration of a partner attachment.
  - `ID` (string, required): ID of the partner attachment

- **`digitalocean-partner-attachment-update`**
  Update a partner attachment.
  - `ID` (string, required): ID of the partner attachment
  - `Name` (string, required): New name
  - `VPCIDs` (array of strings, required): VPC IDs to associate

- **`digitalocean-partner-attachment-get`**  
  Get partner attachment information by ID.  
  - `ID` (string, required): ID of the partner attachment

- **`digitalocean-partner-attachment-list`**  
  List partner attachments with pagination.  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Items per page

---

## Example Queries Using Networking MCP Tools

- Create a new domain "example.com" pointing to IP "203.0.113.10".
- Add an A record to "example.com" for "www" pointing to "203.0.113.20".
- Delete the TXT record with ID 12345 from "example.com".
- Create a new SSL certificate for "myapp.com".
- Delete a firewall with ID "abcd-1234".
- Add HTTP and HTTPS inbound rules to firewall "fw-123".
- Remove SSH access rule from firewall "fw-456".
- Reserve a new IPv4 in region "nyc3".
- Assign reserved IP "198.51.100.5" to droplet 987654.
- Create a new VPC named "private-net" in region "sfo2".
- Flush the cache for CDN with ID "cdn-xyz" for file "/static/logo.png".
- Create a partner attachment named "fast-connect" in region "nyc3" with 1000 Mbps bandwidth.

---

## Notes

- All resource identifiers (IDs, names, IPs) must be replaced with actual values in your queries.
- All responses are returned in JSON format for easy parsing and integration.
- For endpoints that require an ID, name, or IP, replace the placeholder with the appropriate value.
- Use the tools to automate and manage all aspects of DigitalOcean networking from domains and DNS to VPCs, firewalls, and advanced partner connectivity.

---

package docs

import (
	"strings"
)

// Ranking priors. Metadata matching alone cannot tell a hand-written guide
// from a generated reference page, a hub page, or a support article, so the
// scorer multiplies its score by a factor derived from the page kind and the
// intent of the query. Factors only reorder pages that already matched.
type queryProfile struct {
	interfaces bool // asks for a CLI, SDK, API or provider reference
	trouble    bool // describes an error or symptom
	action     bool // asks how to do a task
	pricing    bool // asks about cost
	terms      int
}

var interfaceWords = map[string]struct{}{
	"doctl": {}, "pydo": {}, "terraform": {}, "ansible": {}, "godo": {}, "sdk": {}, "cli": {},
	"api": {}, "endpoint": {}, "endpoints": {}, "curl": {}, "permission": {}, "permissions": {},
	"scope": {}, "scopes": {}, "openapi": {}, "provider": {}, "flags": {}, "flag": {}, "command": {},
	"python": {}, "golang": {}, "ruby": {}, "dots": {}, "reference": {},
}

var troubleWords = map[string]struct{}{
	"error": {}, "errors": {}, "fail": {}, "failed": {}, "fails": {}, "failing": {}, "failure": {},
	"cant": {}, "cannot": {}, "unable": {}, "blocked": {}, "denied": {}, "refused": {}, "issue": {},
	"issues": {}, "problem": {}, "problems": {}, "troubleshoot": {}, "troubleshooting": {}, "fix": {},
	"stuck": {}, "broken": {}, "timeout": {}, "why": {}, "crash": {}, "crashing": {}, "slow": {},
	"unreachable": {}, "missing": {}, "not": {}, "wont": {}, "doesnt": {}, "isnt": {},
}

var actionWords = map[string]struct{}{
	"create": {}, "add": {}, "setup": {}, "configure": {}, "enable": {}, "connect": {}, "deploy": {},
	"attach": {}, "mount": {}, "resize": {}, "install": {}, "manage": {}, "assign": {}, "invite": {},
	"push": {}, "download": {}, "customize": {}, "restore": {}, "rebuild": {}, "snapshot": {},
}

var pricingWords = map[string]struct{}{
	"cost": {}, "costs": {}, "price": {}, "prices": {}, "pricing": {}, "billed": {}, "charge": {}, "charges": {},
}

// guideFolders hold task, concept and detail pages; indexFolders are the
// listing pages that sit above them.
var indexFolders = map[string]struct{}{
	"how-to": {}, "details": {}, "getting-started": {}, "concepts": {}, "reference": {}, "tutorials": {},
}

var generatedReferencePrefixes = []string{
	"/reference/doctl/", "/reference/pydo/", "/reference/terraform/", "/reference/ansible/",
	"/reference/api/", "/reference/godo/", "/reference/dots/", "/platform/teams/roles/permissions/",
}

func profileQuery(query string) queryProfile {
	var p queryProfile
	for _, tok := range tokenSplitRe.Split(strings.ToLower(query), -1) {
		if tok == "" {
			continue
		}
		p.terms++
		if _, ok := interfaceWords[tok]; ok {
			p.interfaces = true
		}
		if _, ok := troubleWords[tok]; ok {
			p.trouble = true
		}
		if _, ok := actionWords[tok]; ok {
			p.action = true
		}
		if _, ok := pricingWords[tok]; ok {
			p.pricing = true
		}
		if len(tok) == 3 && tok[0] >= '0' && tok[0] <= '9' {
			p.trouble = true // status codes such as 520
		}
	}
	return p
}

func isHubPath(path string) bool {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) == 1 {
		return true
	}
	if segs[0] == "products" && len(segs) <= 2 {
		return true
	}
	_, ok := indexFolders[segs[len(segs)-1]]
	return ok
}

// priorFactor returns the multiplier for one result.
func priorFactor(path string, p queryProfile, titleCoversQuery bool) float64 {
	f := 1.0
	for _, prefix := range generatedReferencePrefixes {
		if strings.HasPrefix(path, prefix) {
			if !p.interfaces {
				f *= 0.45
			}
			return f
		}
	}
	if strings.HasPrefix(path, "/support/") {
		if !p.trouble {
			f *= 0.65
		}
		return f
	}
	if isHubPath(path) && !titleCoversQuery {
		f *= 0.6
	}
	if p.action && (strings.Contains(path, "/how-to/") || strings.Contains(path, "/getting-started/")) {
		f *= 1.25
	}
	if p.pricing && strings.HasSuffix(path, "/pricing/") {
		f *= 1.6
	}
	return f
}

// coverageFactor rewards pages whose title and final URL segment are fully
// explained by the query, so "create a VPC network" prefers .../how-to/create/
// over .../how-to/create-peering/. Range 0.85 to 1.5.
func coverageFactor(terms []queryTerm, title, slug []queryTerm) float64 {
	// Stop words are skipped: "How to Create Functions" is wholly explained by
	// "create a serverless function", and counting "how" and "to" against it
	// would rate it below a page with a longer, less relevant title.
	cover := func(tokens []queryTerm) float64 {
		hit, total := 0, 0
		for _, tok := range tokens {
			if _, skip := stopWords[tok.text]; skip {
				continue
			}
			total++
			for _, term := range terms {
				if term.matchesToken(tok) {
					hit++
					break
				}
			}
		}
		if total == 0 {
			return 0
		}
		return float64(hit) / float64(total)
	}
	return 0.85 + 0.3*cover(title) + 0.35*cover(slug)
}

// termCoverageFactor discounts a page that explains only part of the query.
// allTermsBoost rewards complete coverage as a step; this is its smooth
// counterpart, so a page matching two of seven terms stops competing with one
// matching six.
func termCoverageFactor(matched, total int) float64 {
	if total == 0 {
		return 1.0
	}
	return 0.6 + 0.4*float64(matched)/float64(total)
}

// synonyms maps a query word to other words the docs use for the same idea,
// so "remove an organization" reaches "How to Delete Organizations". Matching
// any alternate counts as matching the term; the list is deliberately short
// and limited to everyday verbs and nouns, never product names.
var synonyms = map[string][]string{
	"remove": {"delete", "destroy"}, "delete": {"remove", "destroy"}, "destroy": {"delete", "remove"},
	"terminate": {"delete", "destroy"},
	"cost":      {"pricing"}, "costs": {"pricing"}, "price": {"pricing"}, "prices": {"pricing"},
	"charge": {"pricing"}, "charges": {"pricing"}, "billed": {"pricing"},
	"started": {"quickstart"}, "start": {"quickstart"}, "begin": {"quickstart"},
	"turn": {"enable"}, "activate": {"enable"},
	"cross": {"cors"}, "origin": {"cors"},
	"teammate": {"member", "membership"}, "teammates": {"member", "membership"}, "colleague": {"member", "membership"},
	"datacenter": {"region"}, "datacenters": {"region"},
	"add": {"create"}, "create": {"add"}, "make": {"create"},
	"list": {"retrieve"}, "see": {"view"}, "show": {"view"},
}

func synonymsFor(token string) []queryTerm {
	alts := synonyms[token]
	out := make([]queryTerm, len(alts))
	for i, a := range alts {
		out[i] = queryTerm{text: a, stem: lightStem(a)}
	}
	return out
}

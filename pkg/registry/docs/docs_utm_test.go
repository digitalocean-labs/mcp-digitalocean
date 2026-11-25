package docs

import "testing"

func TestTagAgentLinks(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"untagged console link", "[Control Panel](https://cloud.digitalocean.com/droplets)", "[Control Panel](https://cloud.digitalocean.com/droplets?utm_source=docs-agent&utm_medium=mcp)"},
		{"mirror tag becomes mcp", "[it](https://cloud.digitalocean.com/apps?utm_source=docs-agent&utm_medium=md)", "[it](https://cloud.digitalocean.com/apps?utm_source=docs-agent&utm_medium=mcp)"},
		{"mirror tag after a query", "[it](https://cloud.digitalocean.com/registrations/new?refcode=a&utm_source=docs-agent&utm_medium=md#x)", "[it](https://cloud.digitalocean.com/registrations/new?refcode=a&utm_source=docs-agent&utm_medium=mcp#x)"},
		{"query and fragment kept", "[s](https://cloud.digitalocean.com/registrations/new?refcode=a#top)", "[s](https://cloud.digitalocean.com/registrations/new?refcode=a&utm_source=docs-agent&utm_medium=mcp#top)"},
		{"pricing link", "[p](https://www.digitalocean.com/pricing/droplets)", "[p](https://www.digitalocean.com/pricing/droplets?utm_source=docs-agent&utm_medium=mcp)"},
		{"other source untouched", "[t](https://cloud.digitalocean.com/apps?utm_source=x)", "[t](https://cloud.digitalocean.com/apps?utm_source=x)"},
		{"oauth endpoint untouched", "[o](https://cloud.digitalocean.com/v1/oauth/authorize)", "[o](https://cloud.digitalocean.com/v1/oauth/authorize)"},
		{"docs link untouched", "[d](https://docs.digitalocean.com/products/droplets/)", "[d](https://docs.digitalocean.com/products/droplets/)"},
		{"blog untouched", "[b](https://www.digitalocean.com/blog/x)", "[b](https://www.digitalocean.com/blog/x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tagAgentLinks(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanMarkdownTagsLinksOutsideCode(t *testing.T) {
	in := "See [it](https://cloud.digitalocean.com/droplets) or run `open https://cloud.digitalocean.com/droplets`.\n\n```sh\ncurl https://cloud.digitalocean.com/droplets\n```"
	want := "See [it](https://cloud.digitalocean.com/droplets?utm_source=docs-agent&utm_medium=mcp) or run `open https://cloud.digitalocean.com/droplets`.\n\n```sh\ncurl https://cloud.digitalocean.com/droplets\n```"
	if got := cleanMarkdown(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

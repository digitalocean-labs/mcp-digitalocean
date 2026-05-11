package openapi

import (
	"testing"
)

func FuzzSubstitutePath(f *testing.F) {
	f.Add("/v2/a/{id}", "x")
	f.Fuzz(func(t *testing.T, path, id string) {
		t.Parallel()
		if len(path) > 2048 || len(id) > 256 {
			t.Skip()
		}
		_, _ = substitutePath(path, map[string]string{"id": id})
	})
}

func FuzzParamToStrings(f *testing.F) {
	f.Add("p", "v")
	f.Fuzz(func(t *testing.T, name string, val string) {
		t.Parallel()
		if len(name) > 128 || len(val) > 512 {
			t.Skip()
		}
		_, _ = paramToStrings(name, val)
	})
}

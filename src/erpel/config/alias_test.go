package config

import (
	"reflect"
	"testing"
)

var aliasDepsTests = []struct {
	a    Alias
	deps map[string]struct{}
}{
	{
		a:    Alias{"foo", "bar"},
		deps: map[string]struct{}{},
	},
	{
		a: Alias{"foo", "This is just a {{test}}, nothing more"},
		deps: map[string]struct{}{
			"test": struct{}{},
		},
	},
	{
		a: Alias{"x", "{{This}} {{is}} {{just}} {{a}} {{test}}, {{nothing}} {{more}}"},
		deps: map[string]struct{}{
			"This":    struct{}{},
			"is":      struct{}{},
			"just":    struct{}{},
			"a":       struct{}{},
			"test":    struct{}{},
			"nothing": struct{}{},
			"more":    struct{}{},
		},
	},
	{
		a: Alias{"foo", "bar {{baz}} quux {{baz}} x {{y}} foo"},
		deps: map[string]struct{}{
			"baz": struct{}{},
			"y":   struct{}{},
		},
	},
}

func TestAliasDeps(t *testing.T) {
	for i, test := range aliasDepsTests {
		deps := test.a.deps()
		if !reflect.DeepEqual(deps, test.deps) {
			t.Errorf("test %d: wrong dependencies, want:\n%v\n  got:\n%v\n", i, test.deps, deps)
			continue
		}
	}
}

var aliasTests = []struct {
	before []Alias
	after  []Alias
}{
	{
		before: []Alias{
			NewAlias("foo", "bar"),
			NewAlias("bar", "baz"),
			NewAlias("test", "test"),
		},
		after: []Alias{
			NewAlias("foo", "bar"),
			NewAlias("bar", "baz"),
			NewAlias("test", "test"),
		},
	},
	{
		before: []Alias{
			NewAlias("foo", "bar"),
			NewAlias("bar", "fo{{foo}}obar{{foo}}"),
			NewAlias("baz", "quux"),
			NewAlias("test", "resolv{{bar}}-{{baz}}"),
		},
		after: []Alias{
			NewAlias("foo", "bar"),
			NewAlias("bar", "fobarobarbar"),
			NewAlias("baz", "quux"),
			NewAlias("test", "resolvfobarobarbar-quux"),
		},
	},
	{
		before: []Alias{
			NewAlias("IP", "({{IPv4}}|{{IPv6}})"),
			NewAlias("IPv4", `({{octet}}\.){3}\.{{octet}}`),
			NewAlias("octet", `\d{1,3}`),
			NewAlias("IPv6", `([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}`),
		},
		after: []Alias{
			NewAlias("IP", `((\d{1,3}\.){3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})`),
			NewAlias("IPv4", `(\d{1,3}\.){3}\.\d{1,3}`),
			NewAlias("octet", `\d{1,3}`),
			NewAlias("IPv6", `([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}`),
		},
	},
}

func TestResolveAliases(t *testing.T) {
	for i, test := range aliasTests {
		err := resolveAliases(test.before)
		if err != nil {
			t.Errorf("test %d: resolveAliases() returned error: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(test.before, test.after) {
			t.Errorf("test %d: wrong result returned, want:\n  %#v\ngot:\n  %#v", i,
				test.after, test.before)
			continue
		}
	}
}

var invalidAliasTests = [][]Alias{
	{
		NewAlias("foo", "test {{foo}}"),
	},
	{
		NewAlias("bar", "{{foo}}"),
		NewAlias("foo", "test {{bar}}"),
	},
	{
		NewAlias("x", "y"),
		NewAlias("bar", "{{foo}}"),
		NewAlias("foo", "test {{bar}}"),
	},
	{
		NewAlias("foo1", "{{foo2}}"),
		NewAlias("foo2", "{{foo3}}"),
		NewAlias("foo3", "{{foo4}}"),
		NewAlias("foo4", "{{foo1}}"),
	},
	{
		NewAlias("bar", "{{foo}}"),
		NewAlias("foo", "test {{bar}}"),
		NewAlias("bar", "{{xyz}}"),
	},
}

func TestResolveInvalidAliases(t *testing.T) {
	for i, test := range invalidAliasTests {
		err := resolveAliases(test)
		if err == nil {
			t.Errorf("test %v: returned no error for invalid alias list:\n%#v", i, test)
			continue
		}
	}
}

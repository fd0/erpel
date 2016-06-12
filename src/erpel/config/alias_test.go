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
	before map[string]Alias
	after  map[string]Alias
}{
	{
		before: map[string]Alias{
			"foo":  NewAlias("foo", "bar"),
			"bar":  NewAlias("bar", "baz"),
			"test": NewAlias("test", "test"),
		},
		after: map[string]Alias{
			"foo":  NewAlias("foo", "bar"),
			"bar":  NewAlias("bar", "baz"),
			"test": NewAlias("test", "test"),
		},
	},
	{
		before: map[string]Alias{
			"foo":  NewAlias("foo", "bar"),
			"bar":  NewAlias("bar", "fo{{foo}}obar{{foo}}"),
			"baz":  NewAlias("baz", "quux"),
			"test": NewAlias("test", "resolv{{bar}}-{{baz}}"),
		},
		after: map[string]Alias{
			"foo":  NewAlias("foo", "bar"),
			"bar":  NewAlias("bar", "fobarobarbar"),
			"baz":  NewAlias("baz", "quux"),
			"test": NewAlias("test", "resolvfobarobarbar-quux"),
		},
	},
	{
		before: map[string]Alias{
			"IP":    NewAlias("IP", "({{IPv4}}|{{IPv6}})"),
			"IPv4":  NewAlias("IPv4", `({{octet}}\.){3}\.{{octet}}`),
			"octet": NewAlias("octet", `\d{1,3}`),
			"IPv6":  NewAlias("IPv6", `([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}`),
		},
		after: map[string]Alias{
			"IP":    NewAlias("IP", `((\d{1,3}\.){3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})`),
			"IPv4":  NewAlias("IPv4", `(\d{1,3}\.){3}\.\d{1,3}`),
			"octet": NewAlias("octet", `\d{1,3}`),
			"IPv6":  NewAlias("IPv6", `([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}`),
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

var invalidAliasTests = []map[string]Alias{
	{
		"foo": NewAlias("foo", "test {{foo}}"),
	},
	{
		"bar": NewAlias("bar", "{{foo}}"),
		"foo": NewAlias("foo", "test {{bar}}"),
	},
	{
		"x":   NewAlias("x", "y"),
		"bar": NewAlias("bar", "{{foo}}"),
		"foo": NewAlias("foo", "test {{bar}}"),
	},
	{
		"foo1": NewAlias("foo1", "{{foo2}}"),
		"foo2": NewAlias("foo2", "{{foo3}}"),
		"foo3": NewAlias("foo3", "{{foo4}}"),
		"foo4": NewAlias("foo4", "{{foo1}}"),
	},
	{
		"bar": NewAlias("bar", "{{foo}}"),
		"foo": NewAlias("foo", "test {{bar}}"),
		"baz": NewAlias("bar", "{{xyz}}"),
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

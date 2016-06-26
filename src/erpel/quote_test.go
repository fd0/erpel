package erpel

import (
	"reflect"
	"testing"
)

var testUnquoteString = []struct {
	data   string
	result string
}{
	{
		data:   "",
		result: "",
	},
	{
		data:   `"foobar"`,
		result: "foobar",
	},
	{
		data:   `"foo\nbar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\x0abar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\u000abar"`,
		result: "foo\nbar",
	},
	{
		data:   `"foo\"bar"`,
		result: `foo"bar`,
	},
	{
		data:   `'foo bar '`,
		result: "foo bar ",
	},
	{
		data:   `'foo \'bar '`,
		result: "foo 'bar ",
	},
	{
		data:   "`foo'\"bar `",
		result: "foo'\"bar ",
	},
}

func TestUnquoteString(t *testing.T) {
	for i, test := range testUnquoteString {
		s, err := unquoteString(test.data)
		if err != nil {
			t.Errorf("test %d: unquoteString(%q) return error: %v", i, test.data, err)
			continue
		}

		if s != test.result {
			t.Errorf("test %d: unquoteString(%q) return wrong result: want %q, got %q", i, test.data, test.result, s)
			continue
		}
	}
}

var testUnquoteList = []struct {
	data   string
	result []string
}{
	{
		`[]`,
		[]string{},
	},
	{
		`["foo", "bar", 'baz']`,
		[]string{"foo", "bar", "baz"},
	},
	{
		`["f"]`,
		[]string{"f"},
	},
	{
		"['f', `x`]",
		[]string{"f", "x"},
	},
}

func TestUnquoteList(t *testing.T) {
	for i, test := range testUnquoteList {
		res, err := unquoteList(test.data)
		if err != nil {
			t.Errorf("test %d failed: %v (data %q)", i, err, test.data)
			continue
		}

		if !reflect.DeepEqual(test.result, res) {
			t.Errorf("test %d failed: want %#v, got %#v", i, test.result, res)
		}
	}
}

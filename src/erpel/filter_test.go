package erpel

import (
	"regexp"
	"testing"
)

var filterTests = []struct {
	prefix   *regexp.Regexp
	rules    []*regexp.Regexp
	messages []string
	result   []string
}{
	{
		prefix: regexp.MustCompile("^foo: "),
		rules: []*regexp.Regexp{
			regexp.MustCompile("bar"),
		},
		messages: []string{
			"this is a test",
			"foo: bar test message",
			"bar ",
			"foo: other message",
		},
		result: []string{
			"this is a test",
			"bar ",
			"foo: other message",
		},
	},
	{
		prefix: nil,
		rules: []*regexp.Regexp{
			regexp.MustCompile("bar"),
		},
		messages: []string{
			"this is a test",
			"foo: bar test message",
			`message which contains the string "bar" x`,
			"foo: other message",
		},
		result: []string{
			"this is a test",
			"foo: other message",
		},
	},
}

func TestFilter(t *testing.T) {
	for i, test := range filterTests {
		filter := Filter{
			Prefix: test.prefix,
			Rules: test.rules,
		}

		result := filter.Process(test.messages)

		if len(result) != len(test.result) {
			t.Errorf("test %d failed: want %v results, got %v", i, len(test.result), len(result))
		}

		for i := range result {
			if result[i] != test.result[i] {
				t.Errorf("result[%v] does not match: wanted %q, got %q",
					i, result[i], test.result[i])
			}
		}
	}
}

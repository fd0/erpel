package erpel

import (
	"strings"
	"testing"
)

var processTests = []struct {
	rules  []Rules
	data   string
	result string
}{
	{
		data: `foobar
baz

bump
log message`,
		result: `foobar
baz
bump
log message`,
	},
	{
		rules: []Rules{
			Rules{
				Templates: []string{
					"foobar",
				},
			},
		},
		data: `foobar
baz
bump
log message`,
		result: `baz
bump
log message`,
	},
}

func TestProcess(t *testing.T) {
	for i, test := range processTests {
		var res []string
		handler := func(lines []string) error {
			res = append(res, lines...)
			return nil
		}

		err := Process(test.rules, strings.NewReader(test.data), handler)
		if err != nil {
			t.Errorf("test %d failed: %v", i, err)
			continue
		}

		result := strings.Join(res, "\n")

		if result != test.result {
			t.Errorf("test %d: wrong result\n===== want ========\n%s\n===== got =========\n%s\n========\n",
				i, test.result, result)
		}
	}
}

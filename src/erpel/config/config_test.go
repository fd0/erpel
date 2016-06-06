package config

import (
	"reflect"
	"testing"
)

var testConfigFiles = []struct {
	data string
	cfg  Config
}{
	{
		data: `
# load ignore rules from all files in this directory
rules_dir = "/etc/erpel/rules.d"

# prefix must match at the beginning of each line
prefix = ^\w{3} [ :0-9 ]{11} [._[:alnum:]-]+

[aliases]
IP = "(IPv4|IPv6)"
IPv4 = '\d{0,3}\.\d{0,3}\.\d{0,3}\.\d{0,3}'
IPv6 = "([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}"
`,
		cfg: Config{
			RulesDir: "/etc/erpel/rules.d",
			Prefix:   `^\w{3} [ :0-9 ]{11} [._[:alnum:]-]+`,

			Aliases: map[string]string{
				"IP":   "(IPv4|IPv6)",
				"IPv4": `\d{0,3}\.\d{0,3}\.\d{0,3}\.\d{0,3}`,
				"IPv6": "([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4}",
			},
		},
	},
}

func TestParse(t *testing.T) {
	for i, test := range testConfigFiles {
		cfg, err := Parse(test.data)
		if err != nil {
			t.Errorf("test %v: parse failed: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(cfg, test.cfg) {
			t.Errorf("test %v: config is not equal:\n  want:\n    %#v\n  got:\n    %#v", i, test.cfg, cfg)
		}
	}
}

var testUnquoteString = []struct {
	data   string
	result string
}{
	{
		data:   "foobar",
		result: "foobar",
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
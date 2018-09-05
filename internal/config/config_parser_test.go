package config

import (
	"testing"

	erpelRules "github.com/fd0/erpel/internal/rules"
)

var testConfigs = []struct {
	cfg   string
	state State
}{
	{
		cfg: ``,
		state: State{
			Global: map[string]string{},
		},
	},
	{
		cfg: `afoo=  ''  `,
		state: State{
			Global: map[string]string{
				"afoo": "''",
			},
		},
	},
	{
		cfg: "foo= `bar baz  quux`  ",
		state: State{
			Global: map[string]string{
				"foo": "`bar baz  quux`",
			},
		},
	},
	{
		cfg: "foo= `bar'\" baz quux`  ",
		state: State{
			Global: map[string]string{
				"foo": "`bar'\" baz quux`",
			},
		},
	},
	{
		cfg: "foo= `bar\nbaz\t\nquux`  ",
		state: State{
			Global: map[string]string{
				"foo": "`bar\nbaz\t\nquux`",
			},
		},
	},
	{
		cfg: `a="b"`,
		state: State{
			Global: map[string]string{
				"a": `"b"`,
			},
		},
	},
	{
		cfg: `a ="b"  `,
		state: State{
			Global: map[string]string{
				"a": `"b"`,
			},
		},
	},
	{
		cfg: `  x = 'y'`,
		state: State{
			Global: map[string]string{
				"x": "'y'",
			},
		},
	},
	{
		cfg: `a    = 'b='`,
		state: State{
			Global: map[string]string{
				"a": "'b='",
			},
		},
	},
	{
		cfg: `
			foo = "bar"
			baz= 'bumppp'
			`,
		state: State{
			Global: map[string]string{
				"foo": `"bar"`,
				"baz": "'bumppp'",
			},
		},
	},
	{
		cfg: ` foo = "bar"
		# test comment
		baz= "bumppp"
			`,
		state: State{
			Global: map[string]string{
				"foo": `"bar"`,
				"baz": `"bumppp"`,
			},
		},
	},
	{
		cfg: ` foo = "bar baz" `,
		state: State{
			Global: map[string]string{
				"foo": `"bar baz"`,
			},
		},
	},
	{
		cfg: `xx='1'
		yy="2 a oesu saoe ustha osenuthh"
		# comment
		# comment with spaces
		zz="3"
		key ="Value!   "`,
		state: State{
			Global: map[string]string{
				"xx":  "'1'",
				"yy":  `"2 a oesu saoe ustha osenuthh"`,
				"key": `"Value!   "`,
				"zz":  `"3"`,
			},
		},
	},
	{
		cfg: `foo='bar'
		test = "foobar"`,
		state: State{
			Global: map[string]string{
				"foo":  "'bar'",
				"test": `"foobar"`,
			},
		},
	},
	{
		cfg: `test = "foobar"`,
		state: State{
			Global: map[string]string{
				"test": `"foobar"`,
			},
		},
	},
	{
		cfg: `test = "foo\nb\"ar"`,
		state: State{
			Global: map[string]string{
				"test": `"foo\nb\"ar"`,
			},
		},
	},
	{
		cfg: `test = '  foo bar'  `,
		state: State{
			Global: map[string]string{
				"test": `'  foo bar'`,
			},
		},
	},
	{
		cfg: `test = '  foo \'bar'  `,
		state: State{
			Global: map[string]string{
				"test": `'  foo \'bar'`,
			},
		},
	},
	{
		cfg: `Foo-baR_ = "xxy"  `,
		state: State{
			Global: map[string]string{
				"Foo-baR_": `"xxy"`,
			},
		},
	},
	{
		cfg: `Foo_baR = "xxy"  `,
		state: State{
			Global: map[string]string{
				"Foo_baR": `"xxy"`,
			},
		},
	},
	{
		cfg: `foo="bar"
	test = "foobar"
	# comment
	x =   "y! "`,
		state: State{
			Global: map[string]string{
				"foo":  `"bar"`,
				"test": `"foobar"`,
				"x":    `"y! "`,
			},
		},
	},
	{
		cfg: `
		before = 'foobar Test'
		# comment
		name_With_chars = "x"

		`,
		state: State{
			Global: map[string]string{
				"before":          "'foobar Test'",
				"name_With_chars": `"x"`,
			},
		},
	},
	{
		cfg: `global_setting1 = "value1"
	glob_set2 = "foobar"
	# comment
	x =   "y! "

	other_global_vars = 'X'

	`,
		state: State{
			Global: map[string]string{
				"global_setting1":   `"value1"`,
				"glob_set2":         `"foobar"`,
				"x":                 `"y! "`,
				"other_global_vars": "'X'",
			},
		},
	},
	{
		cfg: ``,
		state: State{
			Fields: map[string]erpelRules.Field{},
		},
	},
	{
		cfg: `# comment, nothing more`,
		state: State{
			Fields: map[string]erpelRules.Field{},
		},
	},
	{
		cfg: `field foo {}`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"foo": erpelRules.Field{},
			},
		},
	},
	{
		cfg: `field foo {
	x = "y"
	}`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"foo": erpelRules.Field{
					"x": `"y"`,
				},
			},
		},
	},
	{
		cfg: `field foo {
	x = "y"
	samples = ["a", 'b', 'xxxx']
	}`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"foo": erpelRules.Field{
					"x":       `"y"`,
					"samples": `["a", 'b', 'xxxx']`,
				},
			},
		},
	},
	{
		cfg: `# comment field
	field foo {
	x = "y" # or else
	} #another comment

	# and another
	`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"foo": erpelRules.Field{
					"x": `"y"`,
				},
			},
		},
	},
	{
		cfg: `field f1 { x = "y" }`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"f1": erpelRules.Field{
					"x": `"y"`,
				},
			},
		},
	},
	{
		cfg: `field f1 {
			a = "1"
			b = '2'
	}`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"f1": erpelRules.Field{
					"a": `"1"`,
					"b": `'2'`,
				},
			},
		},
	},
	{
		cfg: `
	field f1 {
	a = "1"
	b = '2'
	}`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"f1": erpelRules.Field{
					"a": `"1"`,
					"b": `'2'`,
				},
			},
		},
	},
	{
		cfg: `
		key1 = "value1"
		field f1 {
			a = "1"
			b = '2'
		}

		key2 = "value2"
		field f2 {
			x = "y"
			y = '..-..'
			z = "foobar"
		} # comment
		`,
		state: State{
			Global: map[string]string{
				"key1": `"value1"`,
				"key2": `"value2"`,
			},
			Fields: map[string]erpelRules.Field{
				"f1": erpelRules.Field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": erpelRules.Field{
					"x": `"y"`,
					"y": `'..-..'`,
					"z": `"foobar"`,
				},
			},
		},
	},
	{
		cfg: `
	# this config file has no Fields
	`,
		state: State{
			Fields: map[string]erpelRules.Field{},
		},
	},
	{
		cfg: `
		# this config is complete
		field f1 {
			a = "1"
			b = '2'
		}

		field f2 {
			x = "y"
			y = '..-..'
			z = "foobar"
		} # comment
	`,
		state: State{
			Fields: map[string]erpelRules.Field{
				"f1": erpelRules.Field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": erpelRules.Field{
					"x": `"y"`,
					"y": `'..-..'`,
					"z": `"foobar"`,
				},
			},
		},
	},
}

func equalMap(t testing.TB, name string, want map[string]string, got map[string]string) {
	var keys []string
	for key := range want {
		keys = append(keys, key)
	}
	for key := range got {
		keys = append(keys, key)
	}

	for _, key := range keys {
		v1, ok := want[key]
		if !ok {
			t.Errorf("%v: extra key %v found\n", name, key)
			continue
		}

		v2, ok := got[key]
		if !ok {
			t.Errorf("%v: missing key %v\n", name, key)
			continue
		}

		if v1 != v2 {
			t.Errorf("%v: values are not equal, want %v, got %v", name, v1, v2)
		}
	}
}

func equalFields(t testing.TB, want map[string]erpelRules.Field, got map[string]erpelRules.Field) {
	var keys []string
	for key := range want {
		keys = append(keys, key)
	}
	for key := range got {
		keys = append(keys, key)
	}

	for _, key := range keys {
		v1, ok := want[key]
		if !ok {
			t.Errorf("extra key %v found\n", key)
			continue
		}

		v2, ok := got[key]
		if !ok {
			t.Errorf("missing key %v\n", key)
			continue
		}

		equalMap(t, "Field "+key, v1, v2)
	}
}

func TestParseConfig(t *testing.T) {
	for i, test := range testConfigs {
		state, err := Parse(test.cfg)
		if err != nil {
			// t.Errorf("config %d: failed to parse: %v", i, err)
			t.Fatalf("config %d: failed to parse: %v", i, err)
			continue
		}

		equalMap(t, "globals", test.state.Global, state.Global)
		equalFields(t, test.state.Fields, state.Fields)
	}
}

var testInvalidConfig = []string{
	`afoo=  `,
	`a=b`,
	` a = b`,
	" a = 'foo\narb'",
	" a = \"foo\narb\"",
}

func TestParseInvalidConfig(t *testing.T) {
	for i, cfg := range testInvalidConfig {
		_, err := Parse(cfg)
		if err == nil {
			t.Errorf("config %d: expected error for invalid config not found, config:\n%q", i, cfg)
			continue
		}
	}
}

var testDebugConfigs = []string{
	"",
	"foo='bar'\n",
	"foo='bar'",
	"foo = 'bar'",
	"foo = 'bar' # comment text",
	"\n",
	`foo = 'bar'
	baz= "bump"`,
	`
	foo = "foo"
	bar = "baz" #comment
	`,
	`
foo = "bar"
field test { }
	`,
	`
foo = "bar"
field test {
# comment
}
	`,
	`
	foo = "bar"
	field test {
		# comment
		foo = 'baz'
	}
	`,
	`# comment field
field foo {
	x = "y" # or else
} #another comment

# and another
`,
	`
# this config is complete
field f1 {
	a = "1"
	b = '2'
}

field f2 {
	x = "y"
	y = '..-..'
	z = "foobar"
} # comment`,
}

func TestDebugConfigParser(t *testing.T) {
	for i, cfg := range testDebugConfigs {
		_, err := Parse(cfg)
		if err != nil {
			t.Fatalf("config %d: failed to parse: %v", i, err)
			continue
		}
	}
}

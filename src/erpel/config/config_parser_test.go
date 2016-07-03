package config

import "testing"

var testConfigs = []struct {
	cfg   string
	state State
}{
	{
		cfg: ``,
		state: State{
			Sections: map[string]Section{
				"": Section{},
			},
		},
	},
	{
		cfg: `afoo=  ''  `,
		state: State{
			Sections: map[string]Section{
				"": Section{"afoo": "''"},
			},
		},
	},
	{
		cfg: "foo= `bar baz  quux`  ",
		state: State{
			Sections: map[string]Section{
				"": Section{"foo": "`bar baz  quux`"},
			},
		},
	},
	{
		cfg: "foo= `bar'\" baz quux`  ",
		state: State{
			Sections: map[string]Section{
				"": Section{"foo": "`bar'\" baz quux`"},
			},
		},
	},
	{
		cfg: "foo= `bar\nbaz\t\nquux`  ",
		state: State{
			Sections: map[string]Section{
				"": Section{"foo": "`bar\nbaz\t\nquux`"},
			},
		},
	},
	{
		cfg: `a="b"`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"a": `"b"`,
				},
			},
		},
	},
	{
		cfg: `a ="b"  `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"a": `"b"`,
				},
			},
		},
	},
	{
		cfg: `  x = 'y'`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"x": "'y'",
				},
			},
		},
	},
	{
		cfg: `a    = 'b='`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"a": "'b='",
				},
			},
		},
	},
	{
		cfg: `
			foo = "bar"
			baz= 'bumppp'
			`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"foo": `"bar"`,
					"baz": "'bumppp'",
				},
			},
		},
	},
	{
		cfg: ` foo = "bar"
		# test comment
		baz= "bumppp"
			`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"foo": `"bar"`,
					"baz": `"bumppp"`,
				},
			},
		},
	},
	{
		cfg: ` foo = "bar baz" `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"foo": `"bar baz"`,
				},
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
			Sections: map[string]Section{
				"": Section{
					"xx":  "'1'",
					"yy":  `"2 a oesu saoe ustha osenuthh"`,
					"key": `"Value!   "`,
					"zz":  `"3"`,
				},
			},
		},
	},
	{
		cfg: `foo='bar'
		test = "foobar"`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"foo":  "'bar'",
					"test": `"foobar"`,
				},
			},
		},
	},
	{
		cfg: `test = "foobar"`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"test": `"foobar"`,
				},
			},
		},
	},
	{
		cfg: `test = "foo\nb\"ar"`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"test": `"foo\nb\"ar"`,
				},
			},
		},
	},
	{
		cfg: `test = '  foo bar'  `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"test": `'  foo bar'`,
				},
			},
		},
	},
	{
		cfg: `test = '  foo \'bar'  `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"test": `'  foo \'bar'`,
				},
			},
		},
	},
	{
		cfg: `Foo-baR_ = "xxy"  `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"Foo-baR_": `"xxy"`,
				},
			},
		},
	},
	{
		cfg: `Foo_baR = "xxy"  `,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"Foo_baR": `"xxy"`,
				},
			},
		},
	},
	{
		cfg: `foo="bar"
	test = "foobar"
	# comment
	x =   "y! "`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"foo":  `"bar"`,
					"test": `"foobar"`,
					"x":    `"y! "`,
				},
			},
		},
	},
	{
		cfg: `foo{}`,
		state: State{
			Sections: map[string]Section{
				"":    Section{},
				"foo": Section{},
			},
		},
	},
	{
		cfg: `foo {
			bar_2 = 'baz'
			quux = 'fump'
		}`,
		state: State{
			Sections: map[string]Section{
				"": Section{},
				"foo": Section{
					"bar_2": "'baz'",
					"quux":  "'fump'",
				},
			},
		},
	},
	{
		cfg: `foo {
		}

		# bit of space after the section

		`,
		state: State{
			Sections: map[string]Section{
				"":    Section{},
				"foo": Section{},
			},
		},
	},
	{
		cfg: `foo {bar = 'baz'
		}`,
		state: State{
			Sections: map[string]Section{
				"": Section{},
				"foo": Section{
					"bar": "'baz'",
				},
			},
		},
	},
	{
		cfg: `
		before = 'foobar Test'
		# comment
		name_With_chars = "x"

		foo {
			bar = "baz"
			other_var = 'config' # comment after value
		}`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"before":          "'foobar Test'",
					"name_With_chars": `"x"`,
				},
				"foo": Section{
					"bar":       `"baz"`,
					"other_var": "'config'",
				},
			},
		},
	},
	{
		cfg: `global_setting1 = "value1"
glob_set2 = "foobar"
# comment
x =   "y! "

# introduce another section
local_rules {
	loc_set1 = 'v1'
	loc_set2 = "this is just a test"
}

other_global_vars = 'X'

	`,
		state: State{
			Sections: map[string]Section{
				"": Section{
					"global_setting1":   `"value1"`,
					"glob_set2":         `"foobar"`,
					"x":                 `"y! "`,
					"other_global_vars": "'X'",
				},
				"local_rules": Section{
					"loc_set1": "'v1'",
					"loc_set2": `"this is just a test"`,
				},
			},
		},
	},
}

func TestParseConfig(t *testing.T) {
	for i, test := range testConfigs {
		state, err := Parse(test.cfg)
		if err != nil {
			t.Errorf("config %d: failed to parse: %v", i, err)
			continue
		}

		for secName, section := range test.state.Sections {
			// t.Logf("test %v: got Sections:\n%#v", i, state.Sections)

			sec, ok := state.Sections[secName]
			if !ok {
				t.Errorf("test %v: section %q not found in parsed result", i, secName)
				continue
			}

			for key, v1 := range section {
				v2, ok := sec[key]
				if !ok {
					t.Errorf("test %v: missing statement %q in state parsed from config", i, key)
					continue
				}

				if v1 != v2 {
					t.Errorf("test %v: wrong value for %q: want %q, got %q", i, key, v1, v2)
				}
			}

			for key, value := range sec {
				if _, ok := section[key]; !ok {
					t.Errorf("test %v: unexpected statement %q found in section %q (value is %q)", i, key, secName, value)
				}
			}
		}

		for secName := range state.Sections {
			_, ok := test.state.Sections[secName]
			if !ok {
				t.Errorf("test %v: unexpected section %q found in parsed result", i, secName)
			}
		}
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

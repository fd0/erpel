package config

import "testing"

var testConfigs = []struct {
	cfg   string
	state configState
}{
	{
		cfg: ``,
		state: configState{
			sections: map[string]section{
				"": section{},
			},
		},
	},
	{
		cfg: `afoo=   `,
		state: configState{
			sections: map[string]section{
				"": section{"afoo": ""},
			},
		},
	},
	{
		cfg: `a=b`,
		state: configState{
			sections: map[string]section{
				"": section{
					"a": "b",
				},
			},
		},
	},
	{
		cfg: `a =b  `,
		state: configState{
			sections: map[string]section{
				"": section{
					"a": "b",
				},
			},
		},
	},
	{
		cfg: `  x = y`,
		state: configState{
			sections: map[string]section{
				"": section{
					"x": "y",
				},
			},
		},
	},
	{
		cfg: `a    = b=`,
		state: configState{
			sections: map[string]section{
				"": section{
					"a": "b=",
				},
			},
		},
	},
	{
		cfg: `
			foo = bar
			baz= bumppp
			`,
		state: configState{
			sections: map[string]section{
				"": section{
					"foo": "bar",
					"baz": "bumppp",
				},
			},
		},
	},
	{
		cfg: ` foo = bar
		# test comment
		baz= bumppp
			`,
		state: configState{
			sections: map[string]section{
				"": section{
					"foo": "bar",
					"baz": "bumppp",
				},
			},
		},
	},
	{
		cfg: ` foo = bar baz `,
		state: configState{
			sections: map[string]section{
				"": section{
					"foo": "bar baz",
				},
			},
		},
	},
	{
		cfg: `xx=1
		yy=2 a oesu saoe ustha osenuthh
		# comment
		# comment with spaces
		zz=3
		key =Value!    `,
		state: configState{
			sections: map[string]section{
				"": section{
					"xx":  "1",
					"yy":  "2 a oesu saoe ustha osenuthh",
					"key": "Value!",
					"zz":  "3",
				},
			},
		},
	},
	{
		cfg: `foo=bar
		test = foobar`,
		state: configState{
			sections: map[string]section{
				"": section{
					"foo":  "bar",
					"test": "foobar",
				},
			},
		},
	},
	{
		cfg: `test = "foobar"`,
		state: configState{
			sections: map[string]section{
				"": section{
					"test": `"foobar"`,
				},
			},
		},
	},
	{
		cfg: `test = "foo\nb\"ar"`,
		state: configState{
			sections: map[string]section{
				"": section{
					"test": `"foo\nb\"ar"`,
				},
			},
		},
	},
	{
		cfg: `test = '  foo bar'  `,
		state: configState{
			sections: map[string]section{
				"": section{
					"test": `'  foo bar'`,
				},
			},
		},
	},
	{
		cfg: `test = '  foo \'bar'  `,
		state: configState{
			sections: map[string]section{
				"": section{
					"test": `'  foo \'bar'`,
				},
			},
		},
	},
	{
		cfg: `Foo-baR_ = "xxy"  `,
		state: configState{
			sections: map[string]section{
				"": section{
					"Foo-baR_": `"xxy"`,
				},
			},
		},
	},
	{
		cfg: `Foo_baR = "xxy"  `,
		state: configState{
			sections: map[string]section{
				"": section{
					"Foo_baR": `"xxy"`,
				},
			},
		},
	},
	{
		cfg: `foo=bar
	test = "foobar"
	# comment
	x =   "y! "`,
		state: configState{
			sections: map[string]section{
				"": section{
					"foo":  "bar",
					"test": `"foobar"`,
					"x":    `"y! "`,
				},
			},
		},
	},
	{
		cfg: `foo{}`,
		state: configState{
			sections: map[string]section{
				"":    section{},
				"foo": section{},
			},
		},
	},
	{
		cfg: `foo {
			bar_2 = baz
			quux = fump
		}`,
		state: configState{
			sections: map[string]section{
				"": section{},
				"foo": section{
					"bar_2": "baz",
					"quux":  "fump",
				},
			},
		},
	},
	{
		cfg: `foo {
		}

		# bit of space after the section

		`,
		state: configState{
			sections: map[string]section{
				"":    section{},
				"foo": section{},
			},
		},
	},
	{
		cfg: `foo {bar = baz
		}`,
		state: configState{
			sections: map[string]section{
				"": section{},
				"foo": section{
					"bar": "baz",
				},
			},
		},
	},
	{
		cfg: `
		before = foobar Test
		# comment
		name_With_chars = x

		foo {
			bar = baz
			other_var = 'config'
		}`,
		state: configState{
			sections: map[string]section{
				"": section{
					"before":          "foobar Test",
					"name_With_chars": "x",
				},
				"foo": section{
					"bar":       "baz",
					"other_var": "'config'",
				},
			},
		},
	},
	{
		cfg: `global_setting1 = value1
glob_set2 = "foobar"
# comment
x =   "y! "

# introduce another section
local_rules {
	loc_set1 = v1
	loc_set2 = "this is just a test"
}

other_global_vars = X

	`,
		state: configState{
			sections: map[string]section{
				"": section{
					"global_setting1":   "value1",
					"glob_set2":         `"foobar"`,
					"x":                 `"y! "`,
					"other_global_vars": "X",
				},
				"local_rules": section{
					"loc_set1": "v1",
					"loc_set2": `"this is just a test"`,
				},
			},
		},
	},
}

func TestParseConfig(t *testing.T) {
	for i, test := range testConfigs {
		state, err := parseConfig(test.cfg)
		if err != nil {
			t.Errorf("config %d: failed to parse: %v", i, err)
			continue
		}

		for secName, section := range test.state.sections {
			// t.Logf("test %v: got sections:\n%#v", i, state.sections)

			sec, ok := state.sections[secName]
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

		for secName := range state.sections {
			_, ok := test.state.sections[secName]
			if !ok {
				t.Errorf("test %v: unexpected section %q found in parsed result", i, secName)
			}
		}
	}
}

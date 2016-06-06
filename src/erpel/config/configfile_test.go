package config

import "testing"

var testConfigs = []struct {
	cfg   string
	state configState
}{
	{
		cfg: ``,
	},
	{
		cfg: `afoo=   `,
		state: configState{
			stmts: map[string]string{
				"afoo": "",
			},
		},
	},
	{
		cfg: `a=b`,
		state: configState{
			stmts: map[string]string{
				"a": "b",
			},
		},
	},
	{
		cfg: `a =b  `,
		state: configState{
			stmts: map[string]string{
				"a": "b",
			},
		},
	},
	{
		cfg: `  x = y`,
		state: configState{
			stmts: map[string]string{
				"x": "y",
			},
		},
	},
	{
		cfg: `a    = b=`,
		state: configState{
			stmts: map[string]string{
				"a": "b=",
			},
		},
	},
	{
		cfg: `
		foo = bar
		baz= bumppp
		`,
		state: configState{
			stmts: map[string]string{
				"foo": "bar",
				"baz": "bumppp",
			},
		},
	},
	{
		cfg: ` foo = bar
	# test comment
	baz= bumppp
		`,
		state: configState{
			stmts: map[string]string{
				"foo": "bar",
				"baz": "bumppp",
			},
		},
	},
	{
		cfg: ` foo = bar baz `,
		state: configState{
			stmts: map[string]string{
				"foo": "bar baz",
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
			stmts: map[string]string{
				"xx":  "1",
				"yy":  "2 a oesu saoe ustha osenuthh",
				"key": "Value!",
				"zz":  "3",
			},
		},
	},
	{
		cfg: `foo=bar
	test = foobar`,
		state: configState{
			stmts: map[string]string{
				"foo":  "bar",
				"test": "foobar",
			},
		},
	},
	{
		cfg: `test = "foobar"`,
		state: configState{
			stmts: map[string]string{
				"test": `"foobar"`,
			},
		},
	},
	{
		cfg: `test = "foo\nb\"ar"`,
		state: configState{
			stmts: map[string]string{
				"test": `"foo\nb\"ar"`,
			},
		},
	},
	{
		cfg: `test = '  foo bar'  `,
		state: configState{
			stmts: map[string]string{
				"test": `'  foo bar'`,
			},
		},
	},
	{
		cfg: `test = '  foo \'bar'  `,
		state: configState{
			stmts: map[string]string{
				"test": `'  foo \'bar'`,
			},
		},
	},
	{
		cfg: `foo=bar
test = "foobar"
# comment
x =   "y! "`,
		state: configState{
			stmts: map[string]string{
				"foo":  "bar",
				"test": `"foobar"`,
				"x":    `"y! "`,
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

		for key, v1 := range test.state.stmts {
			v2, ok := state.stmts[key]
			if !ok {
				t.Errorf("test %v: missing statement %q in state parsed from config", i, key)
				continue
			}

			if v1 != v2 {
				t.Errorf("test %v: wrong value for %q: want %q, got %q", i, key, v1, v2)
			}
		}

		for key, value := range state.stmts {
			if _, ok := test.state.stmts[key]; !ok {
				t.Errorf("test %v: unexpected statement %q found in parsed state (value is %q)", i, key, value)
			}
		}
	}

}

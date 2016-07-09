package rules

import "testing"

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

func equalFields(t testing.TB, want map[string]Field, got map[string]Field) {
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

var testRuleConfigs = []struct {
	cfg   string
	state State
}{
	{
		cfg: ``,
		state: State{
			Fields: map[string]Field{},
		},
	},
	{
		cfg: `prefix = "foo"`,
		state: State{
			Fields: map[string]Field{},
			Options: map[string]string{
				"prefix": `"foo"`,
			},
		},
	},
	{
		cfg: `# comment, nothing more`,
		state: State{
			Fields: map[string]Field{},
		},
	},
	{
		cfg: `field foo {}`,
		state: State{
			Fields: map[string]Field{
				"foo": Field{},
			},
		},
	},
	{
		cfg: `field foo {
	x = "y"
}`,
		state: State{
			Fields: map[string]Field{
				"foo": Field{
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
			Fields: map[string]Field{
				"foo": Field{
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
			Fields: map[string]Field{
				"foo": Field{
					"x": `"y"`,
				},
			},
		},
	},
	{
		cfg: `field f1 { x = "y" }`,
		state: State{
			Fields: map[string]Field{
				"f1": Field{
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
			Fields: map[string]Field{
				"f1": Field{
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
			Fields: map[string]Field{
				"f1": Field{
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
}

field f2 {
	x = "y"
	y = '..-..'
	z = "foobar"
} # comment
`,
		state: State{
			Fields: map[string]Field{
				"f1": Field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": Field{
					"x": `"y"`,
					"y": `'..-..'`,
					"z": `"foobar"`,
				},
			},
		},
	},
	{
		cfg: `
		---
`,
		state: State{
			Fields: map[string]Field{},
		},
	},
	{
		cfg: `
# this config file has no Fields
---

# just some template lines
line 1
line 2
field {
	foo = bar
}

# trailing comment
`,
		state: State{
			Fields: map[string]Field{},
			Templates: []string{
				"line 1",
				"line 2",
				"field {",
				"foo = bar",
				"}",
			},
		},
	},
	{
		cfg: `
		---
	---
`,
		state: State{
			Fields: map[string]Field{},
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

---

# just some template lines
line 1
line 2
field {
	foo = bar
}

# trailing comment

 ------
sample line 1
 # and some more sample lines
sample line 2....
`,
		state: State{
			Fields: map[string]Field{
				"f1": Field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": Field{
					"x": `"y"`,
					"y": `'..-..'`,
					"z": `"foobar"`,
				},
			},
			Templates: []string{
				"line 1",
				"line 2",
				"field {",
				"foo = bar",
				"}",
			},
			Samples: []string{
				"sample line 1",
				"sample line 2....",
			},
		},
	},
	{
		cfg: `
# this config file has no Fields
---

# just some template lines
line 1
line 2
field {
	foo = bar
}

# trailing comment

 ------
sample line 1
 # and some more sample lines
sample line 2....
`,
		state: State{
			Fields: map[string]Field{},
			Templates: []string{
				"line 1",
				"line 2",
				"field {",
				"foo = bar",
				"}",
			},
			Samples: []string{
				"sample line 1",
				"sample line 2....",
			},
		},
	},
}

func TestParseRuleConfig(t *testing.T) {
	for i, test := range testRuleConfigs {
		state, err := Parse(test.cfg)
		if err != nil {
			t.Errorf("config %d: failed to parse: %v", i, err)
			continue
		}

		equalFields(t, test.state.Fields, state.Fields)
		equalMap(t, "Options", test.state.Options, state.Options)

		if len(state.Templates) != len(test.state.Templates) {
			t.Errorf("test %v: unexpected number of template lines returned: want %d, got %d",
				i, len(test.state.Templates), len(state.Templates))

			continue
		}

		for j := range test.state.Templates {
			if test.state.Templates[j] != state.Templates[j] {
				t.Errorf("test %v: template[%d]: want %q, got %q",
					i, j, test.state.Templates[j], state.Templates[j])
			}
		}

		if len(state.Samples) != len(test.state.Samples) {
			t.Errorf("test %v: unexpected number of sample lines returned: want %d, got %d",
				i, len(test.state.Samples), len(state.Samples))

			continue
		}

		for j := range test.state.Samples {
			if test.state.Samples[j] != state.Samples[j] {
				t.Errorf("test %v: samples[%d]: want %q, got %q",
					i, j, test.state.Samples[j], state.Samples[j])
			}
		}
	}
}

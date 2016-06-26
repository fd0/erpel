package rules

import "testing"

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

		for fieldName, wantField := range test.state.Fields {
			gotField, ok := state.Fields[fieldName]
			if !ok {
				t.Errorf("test %v: field %q not found in parsed result", i, fieldName)
				continue
			}

			for key, v1 := range wantField {
				v2, ok := gotField[key]
				if !ok {
					t.Errorf("test %v: missing statement %q in state parsed from config", i, key)
					continue
				}

				if v1 != v2 {
					t.Errorf("test %v: wrong value for %q: want %q, got %q", i, key, v1, v2)
				}
			}

			for key, value := range gotField {
				if _, ok := wantField[key]; !ok {
					t.Errorf("test %v: unexpected statement %q found in field %q (value is %q)", i, key, fieldName, value)
				}
			}
		}

		for fieldName := range state.Fields {
			_, ok := test.state.Fields[fieldName]
			if !ok {
				t.Errorf("test %v: unexpected field %q found in parsed result", i, fieldName)
			}
		}

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

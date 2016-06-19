package rules

import "testing"

var testRuleConfigs = []struct {
	cfg   string
	state ruleState
}{
	{
		cfg: ``,
		state: ruleState{
			fields: map[string]field{},
		},
	},
	{
		cfg: `# comment, nothing more`,
		state: ruleState{
			fields: map[string]field{},
		},
	},
	{
		cfg: `field foo {}`,
		state: ruleState{
			fields: map[string]field{
				"foo": field{},
			},
		},
	},
	{
		cfg: `field foo {
	x = "y"
}`,
		state: ruleState{
			fields: map[string]field{
				"foo": field{
					"x": `"y"`,
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
		state: ruleState{
			fields: map[string]field{
				"foo": field{
					"x": `"y"`,
				},
			},
		},
	},
	{
		cfg: `field f1 { x = "y" }`,
		state: ruleState{
			fields: map[string]field{
				"f1": field{
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
		state: ruleState{
			fields: map[string]field{
				"f1": field{
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
		state: ruleState{
			fields: map[string]field{
				"f1": field{
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
		state: ruleState{
			fields: map[string]field{
				"f1": field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": field{
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
		state: ruleState{
			fields: map[string]field{},
		},
	},
	{
		cfg: `
# this config file has no fields
---

# just some template lines
line 1
line 2
field {
	foo = bar
}

# trailing comment
`,
		state: ruleState{
			fields: map[string]field{},
			templates: []string{
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
		state: ruleState{
			fields: map[string]field{},
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
		state: ruleState{
			fields: map[string]field{
				"f1": field{
					"a": `"1"`,
					"b": `'2'`,
				},
				"f2": field{
					"x": `"y"`,
					"y": `'..-..'`,
					"z": `"foobar"`,
				},
			},
			templates: []string{
				"line 1",
				"line 2",
				"field {",
				"foo = bar",
				"}",
			},
			samples: []string{
				"sample line 1",
				"sample line 2....",
			},
		},
	},
	{
		cfg: `
# this config file has no fields
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
		state: ruleState{
			fields: map[string]field{},
			templates: []string{
				"line 1",
				"line 2",
				"field {",
				"foo = bar",
				"}",
			},
			samples: []string{
				"sample line 1",
				"sample line 2....",
			},
		},
	},
}

func TestParseRuleConfig(t *testing.T) {
	for i, test := range testRuleConfigs {
		state, err := parseRuleFile(test.cfg)
		if err != nil {
			t.Errorf("config %d: failed to parse: %v", i, err)
			continue
		}

		for fieldName, wantField := range test.state.fields {
			gotField, ok := state.fields[fieldName]
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

		for fieldName := range state.fields {
			_, ok := test.state.fields[fieldName]
			if !ok {
				t.Errorf("test %v: unexpected field %q found in parsed result", i, fieldName)
			}
		}

		if len(state.templates) != len(test.state.templates) {
			t.Errorf("test %v: unexpected number of template lines returned: want %d, got %d",
				i, len(test.state.templates), len(state.templates))

			continue
		}

		for j := range test.state.templates {
			if test.state.templates[j] != state.templates[j] {
				t.Errorf("test %v: template[%d]: want %q, got %q",
					i, j, test.state.templates[j], state.templates[j])
			}
		}

		if len(state.samples) != len(test.state.samples) {
			t.Errorf("test %v: unexpected number of sample lines returned: want %d, got %d",
				i, len(test.state.samples), len(state.samples))

			continue
		}

		for j := range test.state.samples {
			if test.state.samples[j] != state.samples[j] {
				t.Errorf("test %v: samples[%d]: want %q, got %q",
					i, j, test.state.samples[j], state.samples[j])
			}
		}
	}
}

// var testInvalidConfig = []string{
// 	`afoo=  `,
// 	`a=b`,
// 	` a = b`,
// 	" a = 'foo\narb'",
// 	" a = \"foo\narb\"",
// }

// func TestParseInvalidConfig(t *testing.T) {
// 	for i, cfg := range testInvalidConfig {
// 		_, err := parseConfig(cfg)
// 		if err == nil {
// 			t.Errorf("config %d: expected error for invalid config not found, config:\n%q", i, cfg)
// 			continue
// 		}
// 	}
// }

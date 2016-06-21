package rules

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fd0/probe"
)

//go:generate peg rulesfile.peg

// ruleState is the internal state used for parsing a rule file.
type ruleState struct {
	// used to temporarily store values while parsing
	name, value string

	currentField field

	// collection of all fields encountered during parsing
	fields map[string]field

	// all message templates
	templates []string
	// some samples that must match the rules
	samples []string
}

type field map[string]string

func (c *ruleState) newField(name string) {
	name = strings.TrimSpace(name)
	f := make(field)
	c.fields[name] = f
	c.currentField = f
}

func (c *ruleState) set(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	c.currentField[key] = value
}

func (c *ruleState) addTemplate(s string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return
	}
	c.templates = append(c.templates, s)
}

func (c *ruleState) addSample(s string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return
	}
	c.samples = append(c.samples, s)
}

// parseRuleFile returns the state for a configuration.
func parseRuleFile(data string) (ruleState, error) {
	fields := make(map[string]field)

	c := &ruleParser{
		ruleState: ruleState{
			fields: fields,
		},
		Buffer: data,
	}

	c.Init()
	err := c.Parse()
	if err != nil {
		// c.PrintSyntaxTree()
		return ruleState{}, probe.Trace(err, data)
	}
	c.Execute()

	return c.ruleState, nil
}

// ParseRules parses the data as an erpel rule file.
func ParseRules(data string) (Rules, error) {
	state, err := parseRuleFile(data)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	rules, err := parseState(state)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	if err := rules.Check(); err != nil {
		return Rules{}, err
	}

	return rules, nil
}

// ParseRulesFile loads rules from a file and parses it.
func ParseRulesFile(filename string) (Rules, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return Rules{}, err
	}

	return ParseRules(string(buf))
}

// ParseAllRulesFiles loads rules from all files in the directory.
func ParseAllRulesFiles(dir string) (rules []Rules, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		r, err := ParseRulesFile(file)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r)
	}

	return rules, nil
}

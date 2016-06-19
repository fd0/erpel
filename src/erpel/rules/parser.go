package rules

import (
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

// // ParseConfig parses the data as an erpel config file.
// func ParseConfig(data string) (Config, error) {
// 	state, err := parseConfig(data)
// 	if err != nil {
// 		return Config{}, probe.Trace(err)
// 	}

// 	cfg, err := parseState(state)
// 	if err != nil {
// 		return Config{}, probe.Trace(err)
// 	}

// 	return cfg, nil
// }

// // ParseConfigFile loads config data from a file and parses it.
// func ParseConfigFile(filename string) (Config, error) {
// 	buf, err := ioutil.ReadFile(filename)
// 	if err != nil {
// 		return Config{}, err
// 	}

// 	return ParseConfig(string(buf))
// }

// Package rules contains the low-level parser for erpel rules.
package rules

import (
	"strings"

	"github.com/pkg/errors"
)

//go:generate peg rulesfile.peg

// State is the internal state used for parsing a rule file.
type State struct {
	// used to temporarily store values while parsing
	name, value string
	inField     bool

	currentField Field

	// global options
	Options map[string]string

	// collection of all fields encountered during parsing
	Fields map[string]Field

	// all message templates
	Templates []string
	// some samples that must match the rules
	Samples []string
}

// Field is a dynamic part of a message.
type Field map[string]string

func (c *State) newField(name string) {
	name = strings.TrimSpace(name)
	f := make(Field)
	c.Fields[name] = f
	c.currentField = f
}

func (c *State) setField(key, value string) {
	c.currentField[key] = value
}

func (c *State) setOption(key, value string) {
	c.Options[key] = value
}

func (c *State) set(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if c.inField {
		c.setField(key, value)
	} else {
		c.setOption(key, value)
	}
}

func (c *State) addTemplate(s string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return
	}
	c.Templates = append(c.Templates, s)
}

func (c *State) addSample(s string) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return
	}
	c.Samples = append(c.Samples, s)
}

// Parse returns the state for a configuration.
func Parse(data string) (State, error) {
	fields := make(map[string]Field)

	c := &ruleParser{
		State: State{
			Fields:  fields,
			Options: make(map[string]string),
		},
		Buffer: data,
	}

	c.Init()
	err := c.Parse()
	if err != nil {
		// c.PrintSyntaxTree()
		return State{}, errors.WithMessage(err, data)
	}
	c.Execute()

	return c.State, nil
}

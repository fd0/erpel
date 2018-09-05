// Package config contains the low-level configuration file parser.
package config

import (
	"strings"

	erpelRules "github.com/fd0/erpel/internal/rules"
	"github.com/pkg/errors"
)

//go:generate peg configfile.peg

// State is the internal state used for parsing the config file.
type State struct {
	// used to temporarily store values while parsing
	name, value string
	inField     bool

	// global configuration statements
	Global map[string]string

	currentField erpelRules.Field

	// collection of all fields encountered during parsing
	Fields map[string]erpelRules.Field
}

func (c *State) setGlobal(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	c.Global[key] = value
}

func (c *State) newField(name string) {
	name = strings.TrimSpace(name)
	f := make(erpelRules.Field)
	c.Fields[name] = f
	c.currentField = f
}

func (c *State) setField(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	c.currentField[key] = value
}

func (c *State) set(key, value string) {
	if c.inField {
		c.setField(key, value)
		return
	}

	c.setGlobal(key, value)
}

// Parse returns the state for a configuration.
func Parse(data string) (State, error) {
	c := &erpelParser{
		State: State{
			Fields: make(map[string]erpelRules.Field),
			Global: make(map[string]string),
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

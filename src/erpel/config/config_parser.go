// Package config contains the low-level configuration file parser.
package config

import (
	"strings"

	"github.com/fd0/probe"
)

//go:generate peg configfile.peg

// State is the internal state used for parsing the config file.
type State struct {
	// used to temporarily store values while parsing
	name, value string

	currentSection Section

	// collection of all statements encountered during parsing
	Sections map[string]Section
}

// Section contains statements within a section.
type Section map[string]string

func (c *State) newSection(name string) {
	name = strings.TrimSpace(name)
	sec := make(Section)
	c.Sections[name] = sec
	c.currentSection = sec
}

func (c *State) setDefaultSection() {
	c.currentSection = c.Sections[""]
}

func (c *State) set(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	c.currentSection[key] = value
}

// Parse returns the state for a configuration.
func Parse(data string) (State, error) {
	defaultSection := make(Section)
	sections := make(map[string]Section)
	sections[""] = defaultSection

	c := &erpelParser{
		State: State{
			currentSection: defaultSection,
			Sections:       sections,
		},
		Buffer: data,
	}

	c.Init()
	err := c.Parse()
	if err != nil {
		return State{}, probe.Trace(err, data)
	}
	c.Execute()

	return c.State, nil
}

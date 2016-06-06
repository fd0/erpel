package config

import (
	"strings"

	"github.com/fd0/probe"
)

//go:generate peg configfile.peg

// configState is the internal state used for parsing the config file.
type configState struct {
	// used to temporarily store values while parsing
	name, value string

	currentSection section

	// collection of all statements encountered during parsing
	sections map[string]section
}

type section map[string]string

func (c *configState) newSection(name string) {
	name = strings.TrimSpace(name)
	sec := make(section)
	c.sections[name] = sec
	c.currentSection = sec
}

func (c *configState) set(key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	c.currentSection[key] = value
}

// Config holds all configuration from a config file.
type Config struct {
	Statements map[string]string
}

// parseConfig returns the state for a configuration.
func parseConfig(data string) (configState, error) {
	defaultSection := make(section)
	sections := make(map[string]section)
	sections[""] = defaultSection

	c := &erpelParser{
		configState: configState{
			currentSection: defaultSection,
			sections:       sections,
		},
		Buffer: data,
	}

	c.Init()
	err := c.Parse()
	if err != nil {
		c.PrintSyntaxTree()
		return configState{}, probe.Trace(err, data)
	}
	c.Execute()

	return c.configState, nil
}

// parseState returns a Config struct from a state.
func parseState(state configState) (Config, error) {
	return Config{}, nil
}

// Parse parses the data as an erpel config file.
func Parse(data string) (Config, error) {
	state, err := parseConfig(data)
	if err != nil {
		return Config{}, probe.Trace(err)
	}

	cfg, err := parseState(state)
	if err != nil {
		return Config{}, probe.Trace(err)
	}

	return cfg, nil
}

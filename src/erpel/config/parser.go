package config

import "github.com/fd0/probe"

//go:generate peg configfile.peg

// erpelCfg is the internal state for parsing the config file.
type erpelCfg struct {
	name, value string

	stmts map[string]string
}

func (c *erpelCfg) set(key, value string) {
	c.stmts[key] = value
}

// Config holds all configuration from a config file.
type Config struct {
	Statements map[string]string
}

// Parse parses the data as an erpel config file.
func Parse(data string) (Config, error) {
	c := &erpelParser{
		erpelCfg: erpelCfg{
			stmts: make(map[string]string),
		},
		Buffer: data,
	}

	c.Init()
	err := c.Parse()
	if err != nil {
		c.PrintSyntaxTree()
		return Config{}, probe.Trace(err, data)
	}
	c.Execute()

	return Config{Statements: c.stmts}, nil
}

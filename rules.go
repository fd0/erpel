package main

import "github.com/fd0/erpel/internal/erpel"

// Rules contain the ignore rules for log messages.
var Rules []erpel.Rules

// LoadRules loads the rules from the directory and parses the files.
func LoadRules() error {
	V("load rules from %v\n", rulesDir)

	rules, err := erpel.ParseAllRulesFiles(cfg.Fields, rulesDir)
	if err != nil {
		return err
	}

	Rules = rules

	V("loaded rules from %d files\n", len(rules))

	return nil
}

package main

import "erpel"

// Rules contain the ignore rules for log messages.
var Rules []erpel.Rules

var rulesDir string

func init() {
	processCmd.PersistentFlags().StringVarP(&rulesDir, "rules", "r", "/etc/erpel/rules.d", "load rules from this directory")
}

// LoadRules loads the rules from the directory and parses the files.
func LoadRules() error {
	V("load rules from %v\n", rulesDir)

	rules, err := erpel.ParseAllRulesFiles(rulesDir)
	if err != nil {
		return err
	}

	Rules = rules

	V("loaded rules from %d files\n", len(rules))

	return nil
}

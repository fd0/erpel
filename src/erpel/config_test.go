package erpel

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "config_test_*.hcl"))
	if err != nil {
		t.Fatalf("error loading sample config files: %v", err)
	}

	for _, filename := range files {
		_, err := LoadConfig(filename)
		if err != nil {
			t.Errorf("loading config from file %v failed: %v", filename, err)
		}
	}
}

func TestLoadRules(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "rules_test_*.hcl"))
	if err != nil {
		t.Fatalf("error loading sample rules files: %v", err)
	}

	for _, filename := range files {
		_, err := LoadRules(filename)
		if err != nil {
			t.Errorf("loading rules from file %v failed: %v", filename, err)
		}
	}
}

func TestLoadInvalidRules(t *testing.T) {
	rules, err := LoadRules(filepath.Join("testdata", "rules_invalid.hcl"))
	if err == nil {
		t.Fatalf("expected load error not found, got nil")
	}

	if rules != nil {
		t.Fatalf("non-nil return value for invalid rules returned")
	}
}

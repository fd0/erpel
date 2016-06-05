package erpel

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "config_test_*.conf"))
	if err != nil {
		t.Fatalf("error loading sample config files: %v", err)
	}

	for _, filename := range files {
		_, err := LoadConfigFile(filename)
		if err != nil {
			t.Errorf("loading config from file %v failed: %v", filename, err)
		}
	}
}

// func TestLoadRules(t *testing.T) {
// 	cfg := &Config{}

// 	files, err := filepath.Glob(filepath.Join("testdata", "rules_test_*.conf"))
// 	if err != nil {
// 		t.Fatalf("error loading sample rules files: %v", err)
// 	}

// 	for _, filename := range files {
// 		_, err := LoadRules(cfg, filename)
// 		if err != nil {
// 			t.Errorf("loading rules from file %v failed: %v", filename, err)
// 		}
// 	}
// }

// func TestLoadInvalidRules(t *testing.T) {
// 	cfg := &Config{}
// 	rules, err := LoadRules(cfg, filepath.Join("testdata", "rules_invalid.conf"))
// 	if err == nil {
// 		t.Fatalf("expected load error not found, got nil")
// 	}

// 	if rules != nil {
// 		t.Fatalf("non-nil return value for invalid rules returned")
// 	}
// }

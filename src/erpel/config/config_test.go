package config

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

var testConfigFiles = []struct {
	data string
	cfg  Config
}{
	{
		data: `
# load ignore rules from all files in this directory
rules_dir = "/etc/erpel/rules.d"
`,
		cfg: Config{
			RulesDir: "/etc/erpel/rules.d",
		},
	},
}

func TestParse(t *testing.T) {
	for i, test := range testConfigFiles {
		cfg, err := ParseConfig(test.data)
		if err != nil {
			t.Errorf("test %v: parse failed: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(cfg, test.cfg) {
			t.Errorf("test %v: config is not equal:\n  want:\n    %#v\n  got:\n    %#v", i, test.cfg, cfg)
		}
	}
}

func TestSampleConfig(t *testing.T) {
	buf, err := ioutil.ReadFile(filepath.Join("testdata", "erpel.conf"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseConfig(string(buf))
	if err != nil {
		t.Fatalf("parsing sample config failed: %v", err)
	}
}

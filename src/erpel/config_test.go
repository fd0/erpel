package erpel

import (
	"erpel/config"
	"io/ioutil"
	"path/filepath"
	"regexp"
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
			Fields:   map[string]Field{},
		},
	},
	{
		data: `
# load ignore rules from all files in this directory
rules_dir = "/etc/erpel/rules.d"

field timestamp { # the timestamp field
    template = 'Jun  2 23:17:13'
    pattern = '\w{3}  ?\d{1,2} \d{2}:\d{2}:\d{2}'
}

field IP {
    template = '1.2.3.4'
    pattern = '(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})'
    samples = ['192.168.100.1', '2003::feff:1234']
}
`,
		cfg: Config{
			RulesDir: "/etc/erpel/rules.d",
			Fields: map[string]Field{
				"timestamp": Field{
					Name:     "timestamp",
					Pattern:  regexp.MustCompile(`\w{3}  ?\d{1,2} \d{2}:\d{2}:\d{2}`),
					Template: "Jun  2 23:17:13",
				},
				"IP": Field{
					Name:     "IP",
					Pattern:  regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|([0-9a-f]{0,4}:){0,7}[0-9a-f]{0,4})`),
					Template: "1.2.3.4",
					Samples:  []string{"192.168.100.1", "2003::feff:1234"},
				},
			},
		},
	},
}

func TestParseConfig(t *testing.T) {
	for i, test := range testConfigFiles {
		cfg, err := ParseConfig(test.data)
		if err != nil {
			t.Errorf("test %v: parse failed: %v", i, err)
			continue
		}

		if cfg.RulesDir != test.cfg.RulesDir {
			t.Errorf("rules_dir is incorrect, want %v, got %v", test.cfg.RulesDir, cfg.RulesDir)
		}

		var fields []string
		for name := range test.cfg.Fields {
			fields = append(fields, name)
		}
		for name := range cfg.Fields {
			fields = append(fields, name)
		}

		for _, name := range fields {
			want, ok := test.cfg.Fields[name]
			if !ok {
				t.Errorf("extra field %v found", name)
				continue
			}

			got, ok := cfg.Fields[name]
			if !ok {
				t.Errorf("field %v missing", name)
				continue
			}

			if !want.Equals(got) {
				t.Errorf("field %v has wrong value: want %v, got %v", name, want, got)
			}
		}
	}
}

func TestParseSampleConfig(t *testing.T) {
	buf, err := ioutil.ReadFile(filepath.Join("testdata", "erpel.conf"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = config.Parse(string(buf))
	if err != nil {
		t.Fatalf("parsing sample config failed: %v", err)
	}
}

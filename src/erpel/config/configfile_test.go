package config

import (
	"fmt"
	"testing"
)

var testConfigs = []string{
	``,
	`a=b`,
	`a =b  `,
	`  x = y`,
	`a    = b=`,
	`
foo = bar
baz= bumppp
	`,
	`xx=1
yy=2 a oesu saoe ustha osenuthh
# comment
key =Value!    
# comment with spaces
zz=3
	`,
}

func TestConfigFile(t *testing.T) {
	for i, testConfig := range testConfigs {
		cfg, err := Parse(testConfig)
		if err != nil {
			t.Errorf("config %d: failed to parse failed: %v", i, err)
			continue
		}

		fmt.Printf("statements:\n")
		for k, v := range cfg.Statements {
			fmt.Printf("  %q = %q\n", k, v)
		}
	}

}

package erpel

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fd0/probe"
	"github.com/hashicorp/hcl"
)

// Config configures an erpel instance.
type Config struct {
	RulesDir string `hcl:"rules_dir"`
	Prefix   string `hcl:"prefix"`

	prefix *regexp.Regexp
}

// LoadConfig unmarshals the configuration contained in the file.
func LoadConfig(filename string) (*Config, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, probe.Trace(err, filename)
	}

	cfg := &Config{}
	err = hcl.Unmarshal(buf, cfg)
	if err != nil {
		return nil, probe.Trace(err, string(buf))
	}

	if cfg.Prefix != "" {
		// add begining-of-line matching if not already present
		if cfg.Prefix[0] != '^' {
			cfg.Prefix = "^" + cfg.Prefix
		}

		r, err := regexp.Compile(cfg.Prefix)
		if err != nil {
			return nil, probe.Trace(err, cfg.Prefix)
		}

		cfg.prefix = r
	}

	return cfg, nil
}

// LoadAllRules loads all rules from files in dir.
func LoadAllRules(dir string) (rules []*regexp.Regexp, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		r, err := LoadRules(file)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r...)
	}

	return rules, nil
}

// LoadRules unmarshals a rules file.
func LoadRules(filename string) (rules []*regexp.Regexp, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, probe.Trace(err, filename)
	}

	defer func() {
		e := probe.Trace(f.Close())
		if err == nil {
			err = e
		}
	}()

	sc := bufio.NewScanner(f)
	currentLine := 0

	for sc.Scan() {
		currentLine++
		line := strings.TrimSpace(sc.Text())

		// filter out comments and empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// add beginning-of-line matching if not already present
		if line[0] != '^' {
			line = "^" + line
		}

		r, err := regexp.Compile(line)
		if err != nil {
			return nil, probe.Trace(err, filename, currentLine, sc.Text())
		}

		rules = append(rules, r)
	}

	return rules, nil
}

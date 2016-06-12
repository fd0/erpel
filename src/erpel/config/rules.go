package config

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fd0/probe"
)

// LoadAllRules loads all rules from files in dir.
func LoadAllRules(dir string, aliases map[string]Alias) (rules []*regexp.Regexp, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		r, err := LoadRules(file, aliases)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r...)
	}

	return rules, nil
}

// LoadRules unmarshals a rules file.
func LoadRules(filename string, aliases map[string]Alias) (rules []*regexp.Regexp, err error) {
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

		line = ApplyAliases(aliases, line)

		r, err := regexp.Compile(line)
		if err != nil {
			return nil, probe.Trace(err, filename, currentLine, sc.Text())
		}

		rules = append(rules, r)
	}

	return rules, nil
}

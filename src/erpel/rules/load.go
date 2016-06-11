package rules

import (
	"bufio"
	"erpel/config"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fd0/probe"
)

// LoadAll loads all rules from files in dir.
func LoadAll(dir string, aliases map[string]config.Alias) (rules []*regexp.Regexp, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		r, err := Load(file, aliases)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r...)
	}

	return rules, nil
}

// Load unmarshals a rules file.
func Load(filename string, aliases map[string]config.Alias) (rules []*regexp.Regexp, err error) {
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

		// replace aliases
		for name, alias := range aliases {
			line = strings.Replace(line, "{{"+name+"}}", alias.Value, -1)
		}

		r, err := regexp.Compile(line)
		if err != nil {
			return nil, probe.Trace(err, filename, currentLine, sc.Text())
		}

		rules = append(rules, r)
	}

	return rules, nil
}

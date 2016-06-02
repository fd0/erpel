package erpel

import (
	"regexp"
	"strings"
)

// Filter drops lines matching a set of ignore rules.
type Filter struct {
	Prefix *regexp.Regexp
	Rules  []*regexp.Regexp
}

// matchPrefix returns the suffix and true when the line matches the prefix
// regexp. Otherwise an empty string and false is returned. If prefix is nil,
// the line is returned verbatim.
func matchPrefix(prefix *regexp.Regexp, line string) (string, bool) {
	if prefix == nil {
		return line, true
	}

	match := prefix.FindStringIndex(line)
	if match == nil {
		return "", false
	}

	return line[match[1]:], true
}

// Process returns all messages that do not match any of the ignore rules.
func (f Filter) Process(input []string) (result []string) {
nextInput:
	for _, line := range input {
		suffix, found := matchPrefix(f.Prefix, line)
		if !found {
			result = append(result, line)
			continue
		}

		suffix = strings.TrimSpace(suffix)

		for _, r := range f.Rules {
			if r.MatchString(suffix) {
				continue nextInput
			}
		}

		result = append(result, line)
	}

	return result
}

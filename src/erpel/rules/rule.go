package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/fd0/probe"
)

// Rules holds all information parsed from a rules file.
type Rules struct {
	Fields    map[string]Field
	Templates []string
	Samples   []string
}

// Field is a dynamic section in a log message.
type Field struct {
	Template string
	Pattern  *regexp.Regexp
	Samples  []string
}

// unquoteString handles the different quotation kinds.
func unquoteString(s string) (string, error) {
	if s == "" {
		return s, nil
	}

	if len(s) == 1 {
		return "", probe.Trace(fmt.Errorf("invalid quoted string %q", s), s)
	}

	switch s[0] {
	case '"':
		return strconv.Unquote(s)
	case '\'':
		s = strings.Replace(s[1:len(s)-1], `\'`, `'`, -1)
		return s, nil
	case '`':
		if s[len(s)-1] != '`' {
			return "", probe.Trace(fmt.Errorf("invalid quoted string %q", s), s)
		}

		return s[1 : len(s)-1], nil
	}

	// raw strings
	return s, nil
}

// unquoteList parsers a list of strings.
func unquoteList(s string) (list []string, err error) {
	if len(s) < 2 {
		return nil, probe.Trace(fmt.Errorf("string %q is too short for a list", s))
	}

	first := 0
	last := len(s) - 1

	if s[first] != '[' || s[last] != ']' {
		return nil, probe.Trace(fmt.Errorf("string %q is not a list", s))
	}

	s = s[first+1 : last]

	if s == "" {
		return []string{}, err
	}

	for _, data := range strings.Split(s, ",") {
		data = strings.TrimSpace(data)
		item, err := unquoteString(data)
		if err != nil {
			return nil, probe.Trace(err, data)
		}

		list = append(list, item)
	}

	return list, nil
}

// parseState returns a Config struct from a state.
func parseState(state ruleState) (Rules, error) {
	rules := Rules{
		Fields: make(map[string]Field),
	}

	for name, field := range state.fields {
		var (
			err error
			f   Field
		)

		for key, value := range field {
			switch value[0] {
			case '"', '\'', '`':
				value, err = unquoteString(value)
				if err != nil {
					return Rules{}, probe.Trace(err, value)
				}
			}

			switch key {
			case "pattern":
				r, err := regexp.Compile(value)
				if err != nil {
					return Rules{}, probe.Trace(err, value)
				}
				f.Pattern = r
			case "template":
				f.Template = value
			case "samples":
				f.Samples, err = unquoteList(value)
				if err != nil {
					return Rules{}, probe.Trace(err, value)
				}
			default:
				return Rules{}, probe.Trace(fmt.Errorf("unknown key %q in field %q", key, name))
			}
		}

		rules.Fields[name] = f
	}

	rules.Templates = state.templates
	rules.Samples = state.samples

	return rules, nil
}

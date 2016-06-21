package rules

import (
	"errors"
	"fmt"
	"reflect"
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

	rexs []*regexp.Regexp
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

		if err := f.Check(); err != nil {
			return Rules{}, probe.Trace(err, name, f)
		}

		rules.Fields[name] = f
	}

	rules.Templates = state.templates
	rules.Samples = state.samples

	return rules, nil
}

// regexps returns the rules as a list of regexps. These are cached internally.
func (r *Rules) regexps() (rules []*regexp.Regexp) {
	if r.rexs != nil {
		return r.rexs
	}

	for _, s := range r.Templates {
		s = "^" + regexp.QuoteMeta(s) + "$"

		for _, field := range r.Fields {
			repl := regexp.QuoteMeta(field.Template)
			s = strings.Replace(s, repl, field.Pattern.String(), -1)
		}

		re, err := regexp.Compile(s)
		if err != nil {
			panic(err)
		}

		rules = append(rules, re)
	}

	r.rexs = rules

	return rules
}

// checkPattern tests whether the r matches s completely.
func checkPattern(r *regexp.Regexp, s string) error {
	match := r.FindStringIndex(s)
	if match == nil {
		return probe.Trace(errors.New("pattern does not match template"), r.String(), s)
	}

	if match[0] != 0 {
		return probe.Trace(fmt.Errorf("pattern does not match template at the beginning, match: %q",
			s[match[0]:match[1]]), r.String(), s)
	}

	if match[1] != len(s) {
		return probe.Trace(fmt.Errorf("pattern does not match template at the end, match: %q",
			s[match[0]:match[1]]), r.String(), s)
	}

	return nil
}

// Check returns an error if the field's pattern does not match the template or
// the samples.
func (f *Field) Check() error {
	if err := checkPattern(f.Pattern, f.Template); err != nil {
		return probe.Trace(err)
	}

	for _, sample := range f.Samples {
		if err := checkPattern(f.Pattern, sample); err != nil {
			return probe.Trace(err)
		}
	}

	return nil
}

// Equals returns true iff f equals other.
func (f Field) Equals(other Field) bool {
	if f.Template != other.Template {
		return false
	}

	if !reflect.DeepEqual(f.Samples, other.Samples) {
		return false
	}

	if f.Pattern.String() != other.Pattern.String() {
		return false
	}

	return true
}

// Match tests if a rule matches s.
func (r *Rules) Match(s string) bool {
	for _, rule := range r.regexps() {
		if err := checkPattern(rule, s); err == nil {
			return true
		}
	}

	return false
}

// Check runs self-tests on the Rules, it returns an error if a message in the
// samples section is not matched by the rules.
func (r *Rules) Check() error {
	for _, sample := range r.Samples {
		if !r.Match(sample) {
			return probe.Trace(errors.New("sample message does not match any rules"), sample)
		}
	}

	return nil
}

package erpel

import (
	"erpel/rules"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
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

// parseRuleState returns a Rules from a state.
func parseRuleState(state rules.State) (Rules, error) {
	rules := Rules{
		Fields: make(map[string]Field),
	}

	for name, field := range state.Fields {
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

	rules.Templates = state.Templates
	rules.Samples = state.Samples

	return rules, nil
}

// RegExps returns the rules as a list of regexps. These are cached internally.
func (r *Rules) RegExps() (rules []*regexp.Regexp) {
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

// Match tests if a rule matches s completely.
func (r *Rules) Match(s string) bool {
	for _, rule := range r.RegExps() {
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

// ParseRules parses the data as an erpel rule file.
func ParseRules(data string) (Rules, error) {
	state, err := rules.Parse(data)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	rules, err := parseRuleState(state)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	if err := rules.Check(); err != nil {
		return Rules{}, err
	}

	return rules, nil
}

// ParseRulesFile loads rules from a file and parses it.
func ParseRulesFile(filename string) (Rules, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return Rules{}, err
	}

	return ParseRules(string(buf))
}

// ParseAllRulesFiles loads rules from all files in the directory.
func ParseAllRulesFiles(dir string) (rules []Rules, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		r, err := ParseRulesFile(file)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r)
	}

	return rules, nil
}
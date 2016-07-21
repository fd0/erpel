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
	Prefix string

	Fields       map[string]Field
	GlobalFields map[string]Field
	Templates    []string
	Samples      []string

	rexs      []*regexp.Regexp
	prefixReg *regexp.Regexp
}

// Field is a dynamic section in a log message.
type Field struct {
	Name     string
	Template string
	Pattern  *regexp.Regexp
	Samples  []string
}

func parseField(name string, field rules.Field) (f Field, err error) {
	f = Field{Name: name}

	for key, value := range field {
		switch value[0] {
		case '"', '\'', '`':
			value, err = unquoteString(value)
			if err != nil {
				return f, probe.Trace(err, value)
			}
		}

		switch key {
		case "pattern":
			r, err := regexp.Compile(value)
			if err != nil {
				return f, probe.Trace(err, value)
			}
			f.Pattern = r
		case "template":
			f.Template = value
		case "samples":
			f.Samples, err = unquoteList(value)
			if err != nil {
				return f, probe.Trace(err, value)
			}
		default:
			return f, probe.Trace(fmt.Errorf("unknown key %q in field %q", key, name))
		}
	}

	return f, f.Check()
}

// parseRuleState returns a Rules from a state.
func parseRuleState(global map[string]Field, state rules.State) (r Rules, err error) {
	rules := Rules{
		Fields:       make(map[string]Field),
		GlobalFields: global,
	}

	for key, value := range state.Options {
		v, err := unquoteString(value)
		if err != nil {
			return Rules{}, probe.Trace(err, value)
		}

		switch key {
		case "prefix":
			rules.Prefix = v
		default:
			return Rules{}, probe.Trace(fmt.Errorf("unknown key %q in config", key))
		}
	}

	for name, field := range state.Fields {
		f, err := parseField(name, field)
		if err != nil {
			return Rules{}, probe.Trace(err, name, f)
		}

		rules.Fields[name] = f
	}

	rules.Templates = state.Templates
	rules.Samples = state.Samples

	return rules, nil
}

func applyFields(s string, fields map[string]Field) string {
	for _, field := range fields {
		repl := regexp.QuoteMeta(field.Template)
		s = strings.Replace(s, repl, field.Pattern.String(), -1)
	}

	return s
}

// RegExps returns the rules as a list of regexps. These are cached internally.
func (r *Rules) RegExps() (rules []*regexp.Regexp) {
	if r.rexs != nil {
		return r.rexs
	}

	if r.Prefix != "" && r.prefixReg == nil {
		s := "^" + regexp.QuoteMeta(r.Prefix)
		s = applyFields(r.Prefix, r.Fields)
		s = applyFields(s, r.GlobalFields)
		r.prefixReg = regexp.MustCompile(s)
	}

	for _, s := range r.Templates {
		s = "^" + regexp.QuoteMeta(r.Prefix) + regexp.QuoteMeta(s) + "$"

		// apply local fields, then global
		s = applyFields(s, r.Fields)
		s = applyFields(s, r.GlobalFields)

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
	if f.Name != other.Name {
		return false
	}

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
	// test prefix first
	if r.prefixReg != nil {
		fmt.Printf(" prefix reg: %v\n", r.prefixReg)
		if !r.prefixReg.MatchString(s) {
			fmt.Printf("  prefix reg does not match string %q\n", s)
			return false
		}
	}

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
func ParseRules(global map[string]Field, data string) (Rules, error) {
	state, err := rules.Parse(data)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	rules, err := parseRuleState(global, state)
	if err != nil {
		return Rules{}, probe.Trace(err)
	}

	return rules, nil
}

// ParseRulesFile loads rules from a file and parses it.
func ParseRulesFile(global map[string]Field, filename string) (Rules, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return Rules{}, err
	}

	return ParseRules(global, string(buf))
}

// ParseAllRulesFiles loads rules from all files in the directory.
func ParseAllRulesFiles(global map[string]Field, dir string) (rules []Rules, err error) {
	pattern := filepath.Join(dir, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, probe.Trace(err, pattern)
	}

	for _, file := range matches {
		if strings.HasPrefix(filepath.Base(file), ".") {
			continue
		}

		r, err := ParseRulesFile(global, file)
		if err != nil {
			return nil, probe.Trace(err, file)
		}

		if err = r.Check(); err != nil {
			return nil, probe.Trace(err, file)
		}

		rules = append(rules, r)
	}

	return rules, nil
}

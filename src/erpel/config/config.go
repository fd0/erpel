package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/fd0/probe"
	"github.com/tkrajina/go-reflector/reflector"
)

// Config holds all information parsed from a configuration file.
type Config struct {
	RulesDir string `name:"rules_dir"`
	Prefix   string `name:"global_prefix"`

	Aliases map[string]Alias
}

// fieldForName returns the field matching the name, either directly (via
// strings.ToLower()) or via the tag. If the field is not found, an error is
// returned.
func fieldForName(obj *reflector.Obj, name, tag string) (*reflector.ObjField, error) {
	for _, field := range obj.FieldsAll() {
		if name == strings.ToLower(field.Name()) {
			return &field, nil
		}

		fieldTag, err := field.Tag(tag)
		if err == nil && name == fieldTag {
			return &field, nil
		}
	}

	return nil, fmt.Errorf("field %q not found", name)
}

// unquoteString handles the different quotation kinds.
func unquoteString(s string) (string, error) {
	switch s[0] {
	case '"':
		return strconv.Unquote(s)
	case '\'':
		s = strings.Replace(s[1:len(s)-1], `\'`, `'`, -1)
		return s, nil
	}

	// raw strings
	return s, nil
}

// updateField takes care of updating the given field with the value. The value
// is converted according to the target field's type.
func updateField(field *reflector.ObjField, value string) error {
	switch field.Kind() {
	case reflect.String:
		s, err := unquoteString(value)
		if err != nil {
			return err
		}
		return field.Set(s)
	}

	return field.Set(value)
}

// apply takes the keys in the map and applies them to the object.
func apply(data map[string]string, tag string, target interface{}) error {
	obj := reflector.New(target)
	if !obj.IsPtr() {
		return probe.Trace(errors.New("object is not a pointer"))
	}

	for key, value := range data {
		field, err := fieldForName(obj, key, tag)
		if err != nil {
			return probe.Trace(err, key)
		}

		err = updateField(field, value)
		if err != nil {
			return probe.Trace(err, key, value)
		}
	}

	return nil
}

// applyMap unquotes the strings and applies them to the map.
func applyMap(data map[string]string, m map[string]string) error {
	for key, value := range data {
		value, err := unquoteString(value)
		if err != nil {
			return err
		}

		m[key] = value
	}

	return nil
}

// compileRegexp parses all regexps and stores them in the map.
func compileRegexp(data map[string]string) (map[string]*regexp.Regexp, error) {
	m := make(map[string]*regexp.Regexp, len(data))
	for key, value := range data {
		value, err := unquoteString(value)
		if err != nil {
			return nil, err
		}

		r, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}

		m[key] = r
	}

	return m, nil
}

// parseState returns a Config struct from a state.
func parseState(state configState) (Config, error) {
	cfg := Config{}

	for name, data := range state.sections {
		var err error
		switch name {
		case "":
			err = apply(data, "name", &cfg)
		case "aliases":
			cfg.Aliases, err = parseAliases(data)
		default:
			err = fmt.Errorf("unknown section %v", name)
		}

		if err != nil {
			return Config{}, probe.Trace(err, name)
		}
	}

	return cfg, nil
}

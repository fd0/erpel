package erpel

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"

	"github.com/fd0/erpel/internal/config"
	"github.com/pkg/errors"
	"github.com/tkrajina/go-reflector/reflector"
)

// Config holds all information parsed from a configuration file.
type Config struct {
	Options map[string]string
	Fields  map[string]Field
}

var validOptions = map[string]struct{}{
	"rules_dir": struct{}{},
	"state_dir": struct{}{},
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
		return errors.WithStack(errors.New("object is not a pointer"))
	}

	for key, value := range data {
		field, err := fieldForName(obj, key, tag)
		if err != nil {
			return errors.WithMessage(err, key)
		}

		err = updateField(field, value)
		if err != nil {
			return errors.Errorf("error updating %v to %v: %v", key, value, err)
		}
	}

	return nil
}

// unquoteMap unquotes the strings in the map.
func unquoteMap(data map[string]string) error {
	for key := range data {
		value, err := unquoteString(data[key])
		if err != nil {
			return err
		}

		data[key] = value
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
func parseState(state config.State) (c Config, err error) {
	cfg := Config{
		Options: make(map[string]string),
		Fields:  make(map[string]Field),
	}

	for name, value := range state.Fields {
		f, err := parseField(name, value)
		if err != nil {
			return c, errors.WithStack(err)
		}
		cfg.Fields[name] = f
	}

	for name, value := range state.Global {
		if _, ok := validOptions[name]; !ok {
			return c, errors.WithStack(fmt.Errorf("unknown configuration option %q", name))
		}

		s, err := unquoteString(value)
		if err != nil {
			return c, errors.WithMessage(err, value)
		}
		cfg.Options[name] = s
	}

	return cfg, nil
}

// ParseConfig parses data as an erpel config file.
func ParseConfig(data string) (Config, error) {
	state, err := config.Parse(data)
	if err != nil {
		return Config{}, errors.WithStack(err)
	}

	cfg, err := parseState(state)
	if err != nil {
		return Config{}, errors.WithStack(err)
	}

	return cfg, nil
}

// ParseConfigFile loads a config from a file.
func ParseConfigFile(filename string) (Config, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}

	return ParseConfig(string(buf))
}

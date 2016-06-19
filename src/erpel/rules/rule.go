package rules

import (
	"fmt"
	"regexp"

	"github.com/fd0/probe"
)

// Field collects information about one field to be replaced in message templates.
type Field struct {
	Name      string
	Rule      *regexp.Regexp
	Templates []string
}

// NewField constructs a new field.
func NewField(name, rule string, templates []string) (*Field, error) {
	r, err := regexp.Compile(rule)
	if err != nil {
		return nil, probe.Trace(err, rule)
	}

	for _, template := range templates {
		if !r.MatchString(template) {
			return nil, probe.Trace(fmt.Errorf("template %v does not match regex %v", template, rule))
		}
	}

	return &Field{Name: name, Rule: r, Templates: templates}, nil
}

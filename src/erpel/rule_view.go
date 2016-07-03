package erpel

import (
	"fmt"
	"strings"
)

// RuleView is the semantic representation of a rule, constructed by replacing
// all fields in a template.
type RuleView []fmt.Stringer

// Text is used within a RuleView for verbatim text.
type Text string

func (t Text) String() string {
	return string(t)
}

// FieldView is used within a RuleView for a field.
type FieldView struct {
	S string
	F Field
}

func (fv FieldView) String() string {
	return "[" + fv.F.Name + "]"
}

func applyField(field Field, data RuleView) (result RuleView) {
	for _, item := range data {
		str, ok := item.(Text)
		if !ok {
			result = append(result, item)
			continue
		}

		matches := strings.Split(string(str), field.Template)
		l := len(matches)
		if l == 1 {
			result = append(result, str)
			continue
		}

		for _, s := range matches[:l-1] {
			// if s is the empty string, the template was found at the
			// beginning of the string, so we don't need to add the string
			// itself.
			if s != "" {
				// Otherwise, append the string to result.
				result = append(result, Text(s))
			}
			result = append(result, FieldView{S: field.Template, F: field})
		}

		last := matches[l-1]
		// if last is no the empty string, there is a text segment trailing, so add it.
		if last != "" {
			result = append(result, Text(matches[l-1]))
		}
	}

	return result
}

// View renders a template into a RuleView by applying the rules.
func View(rules Rules, template string) RuleView {
	data := RuleView{Text(template)}

	for _, field := range rules.Fields {
		data = applyField(field, data)
	}

	return data
}

// Views returns all views for all templates of r.
func (r Rules) Views() (views []RuleView) {
	views = make([]RuleView, 0, len(r.Templates))
	for _, t := range r.Templates {
		views = append(views, View(r, t))
	}

	return views
}
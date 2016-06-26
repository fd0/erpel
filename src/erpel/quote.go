package erpel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fd0/probe"
)

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

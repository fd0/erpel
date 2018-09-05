package erpel

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// HandleFunc handles lines than have not been filtered out by any rules.
type HandleFunc func(lines []string) error

// ProcessFile extracts all log messages starting at the marker from the file
// by opening it and calling Process(). Returned is a marker for the last
// position within the file.
func ProcessFile(rules []Rules, filename string, last Marker, fn HandleFunc) (m Marker, err error) {
	var fd *os.File

	fd, err = os.Open(filename)
	if err != nil {
		return Marker{}, err
	}

	if err = last.Seek(fd); err != nil {
		return Marker{}, err
	}

	defer func() {
		e := fd.Close()
		if err != nil {
			err = e
		}
	}()

	err = Process(rules, fd, fn)
	if err != nil {
		return Marker{}, err
	}

	return Position(fd)
}

const handleBatchSize = 20

// Process extracts all log messages from the reader, ignores those matched by
// the rules and hands the remaining lines to f. When f returns an error,
// processing stops and this error is returned. Empty lines are always ignored.
func Process(rules []Rules, rd io.Reader, f HandleFunc) error {
	sc := bufio.NewScanner(rd)

	var resultLines []string

nextLine:
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		// ignore empty lines
		if line == "" {
			continue nextLine
		}

		for _, rule := range rules {
			if rule.Match(line) {
				continue nextLine
			}
		}

		resultLines = append(resultLines, line)

		if len(resultLines) >= handleBatchSize {
			err := f(resultLines)
			if err != nil {
				return err
			}

			resultLines = resultLines[:0]
		}
	}

	if len(resultLines) > 0 {
		return f(resultLines)
	}

	return nil
}

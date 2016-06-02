package probe

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func retError(params ...interface{}) error {
	return fmt.Errorf("params: %v", params)
}

func funcA(foo string, x int) error {
	return Trace(retError(foo, x), foo, x)
}

func funcB(foo string, x int) error {
	return Trace(funcA(foo, x), foo, x)
}

func funcC(foo string, x int) error {
	return Trace(funcB(foo, x), foo, x)
}

func testDefer() (err error) {
	defer func() {
		err = Trace(funcA("test defer", 34), "within defer")
	}()

	return nil
}

func throwPanic() {
	panic("this is paaanic!")
}

func recoverPanic() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Trace(errors.New("catched panic"), r)
		}
	}()

	throwPanic()
	return nil
}

func TestProbeTrace(t *testing.T) {
	errs := []struct {
		err            error
		backtraceLines int
	}{
		{funcA("test", 123), 2},
		{funcB("test", 1234), 3},
		{funcC("test", 12345), 4},
		{testDefer(), 3},
		{recoverPanic(), 2},
	}

	for i, test := range errs {
		if test.err == nil {
			t.Errorf("test %d returned nil error, expected something non-nil", i)
			continue
		}

		err, ok := test.err.(Error)
		if !ok {
			t.Errorf("test %d failed: returned error is not Error, but %#v", i, test.err)
			continue
		}

		bt := err.Backtrace()
		lines := strings.Count(bt, "\n")
		if lines != test.backtraceLines {
			t.Errorf("wrong number of lines returned in backtrace, want %v, got %v\ntrace:\n%v",
				test.backtraceLines, lines, bt)
		}
	}
}

func TestProbeNil(t *testing.T) {
	err := Trace(nil)
	if err != nil {
		t.Errorf("Trace(nil) returned non-nil value: %v", err)
	}
}

func Example() {
	stat := func(filename string) error {
		_, err := os.Lstat(filename)
		return err
	}

	transform := func(filename string) error {
		f := strings.ToLower(filename)
		return Trace(stat(f), f)
	}

	do := func(filename string) error {
		err := transform(filename)
		return Trace(err, filename)
	}

	err := do("Does_Not_Exist.go")
	if e, ok := err.(Error); ok {
		fmt.Print(e.Backtrace())
	}

	// Output:
	// Error: lstat does_not_exist.go: no such file or directory
	// probe/probe_test.go:98 github.com/fd0/probe.Example.func2 [does_not_exist.go]
	// probe/probe_test.go:103 github.com/fd0/probe.Example.func3 [Does_Not_Exist.go]
}

package probe

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Error wraps an error and adds tracing information.
type Error struct {
	Cause       error
	tracePoints []tracePoint
}

// tracePoint stores data at one point in the code.
type tracePoint struct {
	File     string
	Line     int
	Function string
	Data     []interface{}
}

// Trace takes err and wraps it in an Error (if necessary), afterwards the
// trace point is recorded with the data. If err is nil, nothing is recorded
// and nil is returned.
func Trace(err error, data ...interface{}) error {
	if err == nil {
		return nil
	}

	traceErr := Error{
		Cause: err,
	}

	if e, ok := err.(Error); ok {
		traceErr = e
	}

	traceErr.tracePoints = append(traceErr.tracePoints, newtracePoint(2, data...))
	return traceErr
}

func (err Error) Error() string {
	return err.Backtrace()
}

func (err Error) String() string {
	return fmt.Sprintf("<Error caused by %q (%d trace points)>", err.Cause, len(err.tracePoints))
}

// Backtrace returns a printable trace leading to the error.
func (err Error) Backtrace() string {
	stacktrace := fmt.Sprintf("Error: %v\n", err.Cause)

	for _, tp := range err.tracePoints {
		stacktrace += tp.String() + "\n"
	}

	return stacktrace
}

// newtracePoint returns a new trace point annotated with data.
func newtracePoint(skip int, data ...interface{}) tracePoint {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return tracePoint{Data: data}
	}

	functionName := runtime.FuncForPC(pc).Name()
	return tracePoint{
		Data:     data,
		File:     file,
		Line:     line,
		Function: functionName,
	}
}

func (tp tracePoint) String() string {
	filename := filepath.Base(tp.File)
	dirname := filepath.Base(filepath.Dir(tp.File))
	s := fmt.Sprintf("%s/%s:%d %v", dirname, filename, tp.Line, tp.Function)

	if len(tp.Data) > 0 {
		s += fmt.Sprintf(" %v", tp.Data)
	}

	return s
}

package arrange

import "errors"

// DefaultErrorExitCode is used when no exit code could otherwise be
// determined for a non-nil error.
const DefaultErrorExitCode int = 1

// ExitCoder is an optional interface that an error can implement to supply
// an associated exit code with that error.  Useful particularly with fx.ExitCode
// or to determine the process exit code upon an error.
type ExitCoder interface {
	// ExitCode returns the exit code associated with this error.
	ExitCode() int
}

type exitCodeErr struct {
	error
	exitCode int
}

func (ece exitCodeErr) ExitCode() int {
	return ece.exitCode
}

func (ece exitCodeErr) Unwrap() error {
	return ece.error
}

// UseExitCode returns a new error object that associates an existing error
// with an exit code.  The new error will implement ExitCoder and will have
// an Unwrap method as described in the errors package.
//
// If err is nil, this function immediately panics so as not to delay a panic
// until the returned error is used.
func UseExitCode(err error, exitCode int) error {
	if err == nil {
		panic("cannot associate a nil error with an exit code")
	}

	return exitCodeErr{
		error:    err,
		exitCode: exitCode,
	}
}

// ErrorCoder is a strategy type for determining the exit code for an error.
// Note that this strategy will be invoked with a nil error to allow custom
// logic for returning exit codes indicating success.
type ErrorCoder func(error) int

// ExitCodeFor provides a standard way of determining the exit code associated
// with an error.  Logic is applied in the following order:
//
//   - If err implements ExitCoder, that exit code is returned
//   - If coder is not nil, it is invoked to determine the exit code
//   - If err is not nil, DefaultErrorExitCode is returned
//   - If none of the above are true, this function returns zero (0).
func ExitCodeFor(err error, coder ErrorCoder) int {
	var ec ExitCoder
	switch {
	case errors.As(err, &ec):
		return ec.ExitCode()

	case coder != nil:
		return coder(err) // err can be nil

	case err != nil:
		return DefaultErrorExitCode

	default:
		return 0
	}
}

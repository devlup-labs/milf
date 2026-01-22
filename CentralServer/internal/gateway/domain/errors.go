package domain

import "errors"

var (
	// ErrLambdaNotFound is returned when a lambda cannot be found
	ErrLambdaNotFound = errors.New("lambda not found")

	// ErrInvalidRuntime is returned when an unsupported runtime is specified
	ErrInvalidRuntime = errors.New("invalid runtime environment")

	// ErrInvalidRunningMode is returned when an unsupported running mode is specified
	ErrInvalidRunType = errors.New("invalid run type")

	// ErrCompilationFailed is returned when source code compilation fails
	ErrCompilationFailed = errors.New("compilation failed")

	// ErrExecutionFailed is returned when lambda execution fails
	ErrExecutionFailed = errors.New("execution failed")

	// ErrInvalidRequest is returned when the request payload is invalid
	ErrInvalidRequest = errors.New("invalid request")

	// ErrInternalServer is returned for unexpected server errors
	ErrInternalServer = errors.New("internal server error")
)

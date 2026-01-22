package domain

// Validation constraints for Lambda configuration
const (
	// Memory constraints (in MB)
	MinMemoryMB = 64
	MaxMemoryMB = 4096

	// Name constraints
	MinNameLength = 1
	MaxNameLength = 128
)

// ValidateStoreRequest validates the LambdaStoreRequest
func ValidateStoreRequest(req *LambdaStoreRequest) error {
	if req.Name == "" || req.SourceCode == "" {
		return ErrInvalidRequest
	}

	if len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength {
		return ErrInvalidRequest
	}

	if !IsValidRuntime(req.Runtime) {
		return ErrInvalidRuntime
	}

	if !IsValidRunType(req.RunType) {
		return ErrInvalidRunType
	}

	if req.MemoryMB < MinMemoryMB || req.MemoryMB > MaxMemoryMB {
		return ErrInvalidRequest
	}

	return nil
}

// ValidateExecRequest validates the LambdaExecRequest
func ValidateExecRequest(req *LambdaExecRequest) error {
	if req.ReferenceID == "" {
		return ErrInvalidRequest
	}
	return nil
}

// IsValidRuntime checks if the runtime environment is supported
func IsValidRuntime(rt RuntimeEnvironment) bool {
	switch rt {
	case RuntimeGo, RuntimeRust, RuntimePython, RuntimeJavaScript:
		return true
	default:
		return false
	}
}

// IsValidRunningMode checks if the running mode is supported
func IsValidRunType(mode RunType) bool {
	switch mode {
	case RunTypeOnCommand, RunTypePeriodic:
		return true
	default:
		return false
	}
}

// SupportedRuntimes returns all supported runtime environments
func SupportedRuntimes() []RuntimeEnvironment {
	return []RuntimeEnvironment{
		RuntimeGo,
		RuntimeRust,
		RuntimePython,
		RuntimeJavaScript,
	}
}

// SupportedRunningModes returns all supported running modes
func SupportedRunTypes() []RunType {
	return []RunType{
		RunTypeOnCommand,
		RunTypePeriodic,
	}
}

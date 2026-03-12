package domain

type CompilationError struct {
	LambdaID string
	Stage    string //fetch, build, link
	Message  string
}

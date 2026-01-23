package domain
type Compilationerror struct {
	LambdaID string
	Stage    string //fetch, build, link
	Message  string
}
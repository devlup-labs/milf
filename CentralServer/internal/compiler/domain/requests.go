package domain
import "fmt"
type CompilationRequest struct {
	LambdaID     string
	UserID       string
	Runtime      RuntimeType
	SourceFiles  []SourceFile
	Metadata     FunctionMetadata
	RunImmediate bool
}

func (r *CompilationRequest) Validate() error {
	if r.LambdaID == "" {
		return fmt.Errorf("lambdaID cannot be empty")
	}
	if len(r.SourceFiles) == 0 {
		return fmt.Errorf("no source files provided")
	}

	return nil

}

package domain
type RuntimeType string

const (
	RuntimeGo   RuntimeType = "go"
	RuntimeRust RuntimeType = "rust"
	RuntimeC RuntimeType = "c"
	RuntimeCpp RuntimeType = "cpp"
)

type SourceFile struct {
	Path    string
	Content []byte
}

type FunctionMetadata struct {
	Name        string
	MemoryMB    int
	TimeoutSec int
	EntryPoint string
}

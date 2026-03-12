package core

import (
	"central_server/internal/compiler/domain"
	"central_server/internal/compiler/interfaces"
	gwdomain "central_server/internal/gateway/domain"
	gwinterfaces "central_server/internal/gateway/interfaces"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Compiler struct {
	objectStore  interfaces.ObjectStore
	trigger      interfaces.RunTrigger
	queue        *gwdomain.CompilationQueue
	orchestrator gwinterfaces.OrchestratorService
}

func NewCompiler(
	objectStore interfaces.ObjectStore,
	trigger interfaces.RunTrigger,
	queue *gwdomain.CompilationQueue,
	orchestrator gwinterfaces.OrchestratorService,
) *Compiler {
	return &Compiler{
		objectStore:  objectStore,
		trigger:      trigger,
		queue:        queue,
		orchestrator: orchestrator,
	}
}

func newCompilationError(
	lambdaID string,
	stage string,
	err error,
) domain.CompilationError {
	return domain.CompilationError{
		LambdaID: lambdaID,
		Stage:    stage,
		Message:  err.Error(),
	}
}

func (c *Compiler) Compile(lambdaID string) ([]byte, *domain.CompilationError) {

	// ---- FETCH STAGE ----
	req, err := c.objectStore.FetchCompilationRequest(lambdaID)
	if err != nil {
		ce := newCompilationError(lambdaID, "fetch", err)
		return nil, &ce
	}

	// ---- VALIDATE STAGE ----
	if err := req.Validate(); err != nil {
		ce := newCompilationError(req.LambdaID, "validate", err)
		return nil, &ce
	}

	var wasmBytes []byte

	// ---- BUILD STAGE ----
	switch req.Runtime {

	case domain.RuntimeC:
		wasmBytes, err = c.compileC(req)

	case domain.RuntimeGo:
		err = fmt.Errorf("go runtime not implemented yet")

	case domain.RuntimeRust:
		err = fmt.Errorf("rust runtime not implemented yet")

	case domain.RuntimeCpp:
		err = fmt.Errorf("cpp runtime not implemented yet")

	default:
		err = fmt.Errorf("unsupported runtime")
	}

	if err != nil {
		ce := newCompilationError(req.LambdaID, "build", err)
		return nil, &ce
	}

	// ---- STORE WASM STAGE ----
	err = c.objectStore.StoreWasm(req.LambdaID, wasmBytes)
	if err != nil {
		ce := newCompilationError(req.LambdaID, "store", err)
		return nil, &ce
	}

	// ---- STORE METADATA STAGE ----
	meta := req.Metadata
	meta.LambdaRef = req.LambdaID
	meta.UserID = req.UserID
	meta.TriggerImmediate = req.RunImmediate

	err = c.objectStore.StoreMetadata(req.LambdaID, meta)
	if err != nil {
		ce := newCompilationError(req.LambdaID, "store", err)
		return nil, &ce
	}

	// ---- TRIGGER STAGE ----
	if req.RunImmediate {
		err := c.trigger.TriggerRun(req.LambdaID)
		if err != nil {
			ce := newCompilationError(req.LambdaID, "trigger", err)
			return nil, &ce
		}
	}

	return wasmBytes, nil
}

func (c *Compiler) compileC(req domain.CompilationRequest) ([]byte, error) {

	// 1. Find the C source file
	var cFile *domain.SourceFile

	for _, file := range req.SourceFiles {
		if strings.HasSuffix(file.Path, ".c") {
			cFile = &file
			break
		}
	}

	if cFile == nil {
		return nil, errors.New("no C source file found for C runtime")
	}

	// 2. Create isolated temporary directory
	tempDir, err := os.MkdirTemp("", "compiler-c-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// 3. Construct full path for C source file
	cFilePath := filepath.Join(tempDir, cFile.Path)

	// 4. Create parent directories if needed
	err = os.MkdirAll(filepath.Dir(cFilePath), 0755)
	if err != nil {
		return nil, err
	}

	// 5. Write C source code to disk
	err = os.WriteFile(cFilePath, cFile.Content, 0644)
	if err != nil {
		return nil, err
	}

	// 6. Define output WASM file path
	wasmOutputPath := filepath.Join(tempDir, "output.wasm")

	// 7. Run clang to compile C → WASM (WASI)
	cmd := exec.Command(
		"clang",
		"--target=wasm32-wasi",
		"--sysroot=/opt/wasi-sdk/share/wasi-sysroot",
		cFilePath,
		"-O2",
		"-o",
		wasmOutputPath,
	)

	// Capture compiler output (errors included)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(output))
	}

	// 8. Read compiled WASM binary into memory
	wasmBytes, err := os.ReadFile(wasmOutputPath)
	if err != nil {
		return nil, err
	}

	// 9. Compilation successful
	return wasmBytes, nil
}

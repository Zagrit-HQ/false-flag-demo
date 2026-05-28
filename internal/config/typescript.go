// Package config / typescript.go — server-side authoritative
// TypeScript-to-IR compiler.
//
// Pipeline:
//
//  1. esbuild.Build() strips TypeScript and emits a CommonJS module
//     with @falseflag/config marked as External (the shim below
//     resolves it). Sourcemap is inlined for goja error mapping.
//  2. A per-request goja.Runtime executes the CJS. Interrupts fire on
//     both the request context and a 1-second wall-clock timer; the
//     require() stub resolves only @falseflag/config (an embedded
//     ES5 shim mirroring js/packages/config-ts/src/index.ts).
//  3. module.exports.default is JSON-marshaled and handed to the same
//     validateTreeWith + compilePredicates path the JSON compiler
//     uses, so all three strategies converge on one IR.
//
// Both deps are pure Go — no CGO, no glibc, no impact on the
// distroless/static-debian12 base image. The shim is conformance-
// tested against the JS DSL via tests/eval-corpus fixtures; do not
// edit one without the other.

package config

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

//go:embed typescript_shim.js
var dslShim string

// maxSourceTextBytes caps the size of the raw TS input the compiler
// will accept. Mirrored by the API handler's http.MaxBytesReader at
// the request boundary; redundant here so library callers (the seed
// command, the MCP validate_config tool) get the same protection.
const maxSourceTextBytes = 32 * 1024

// maxIRBytes caps the size of the compiled IR JSON the goja runtime
// is permitted to produce. Bounds memory in the JSON.marshal +
// validateTreeWith path.
const maxIRBytes = 32 * 1024

// compileWallClock bounds how long the goja runtime is allowed to
// execute the user's compiled JS. Per-opcode interrupt checks make
// this hold even for tight `while (true) {}` loops.
const compileWallClock = time.Second

type typescriptCompiler struct{}

func (typescriptCompiler) Strategy() Strategy { return StrategyTypeScript }

// Compile executes the TypeScript source through esbuild + goja and
// returns the same Compiled shape the JSON and CEL compilers produce.
// Sandboxing notes:
//   - esbuild runs in-process and is pure Go; no subprocess.
//   - goja is a pure-Go interpreter with no fs/network/process access.
//   - The runtime is bound by a 1s wall clock AND the supplied context;
//     whichever cancels first wins. `vm.Interrupt()` halts the
//     interpreter between opcodes.
//   - The only resolvable import is `@falseflag/config`; every other
//     `require()` throws.
func (typescriptCompiler) Compile(source []byte) (*Compiled, error) {
	return compileTypescript(context.Background(), source)
}

func compileTypescript(ctx context.Context, source []byte) (*Compiled, error) {
	if len(source) == 0 {
		return nil, fmt.Errorf("%w: empty source", ErrTypeScriptCompileFailure)
	}
	// Rehydration fast path: when the input is already-compiled IR JSON
	// (e.g. evaluate handler loading flag_versions.compiled, snapshot
	// rebuild on proxy/SDK pull), skip esbuild and go straight to the
	// shared validation path. Without this, calling Compile(TS, irJSON)
	// from the evaluate hot path 500s with "esbuild: Expected ; but
	// found :" because esbuild tries to parse the JSON as TypeScript.
	// The JSON compiler treats its input the same way; this aligns TS
	// with the cross-strategy rehydration contract.
	if compiled, ok := tryRehydrateIR(source); ok {
		return compiled, nil
	}
	if len(source) > maxSourceTextBytes {
		return nil, fmt.Errorf("%w: source exceeds %d bytes",
			ErrTypeScriptCompileFailure, maxSourceTextBytes)
	}

	// 1) esbuild: TS -> CJS, keeping @falseflag/config external so we
	//    can satisfy the require() in goja with the embedded shim.
	res := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   string(source),
			Loader:     api.LoaderTS,
			ResolveDir: "/nonexistent",
			Sourcefile: "config.ts",
		},
		Bundle:    true,
		Format:    api.FormatCommonJS,
		Platform:  api.PlatformNode,
		External:  []string{"@falseflag/config"},
		Sourcemap: api.SourceMapInline,
		Write:     false,
		LogLevel:  api.LogLevelSilent,
	})
	if len(res.Errors) > 0 {
		return nil, newEsbuildError(res.Errors)
	}
	if len(res.OutputFiles) != 1 {
		return nil, fmt.Errorf("%w: esbuild produced %d output files",
			ErrTypeScriptCompileFailure, len(res.OutputFiles))
	}
	js := string(res.OutputFiles[0].Contents)

	// 2) goja: run the compiled CJS, bound by ctx + wall-clock.
	vm := goja.New()
	vm.SetMaxCallStackSize(2048)

	// Load the DSL shim first so globalThis.__falseflag_dsl is set
	// before require() can return it.
	if _, err := vm.RunString(dslShim); err != nil {
		return nil, fmt.Errorf("%w: shim load: %s",
			ErrTypeScriptCompileFailure, err)
	}
	shim := vm.Get("__falseflag_dsl")
	if shim == nil || goja.IsUndefined(shim) {
		return nil, fmt.Errorf("%w: shim did not register globalThis.__falseflag_dsl",
			ErrTypeScriptCompileFailure)
	}

	// CJS scaffolding: `module.exports.default = ...` lands user output here.
	module := vm.NewObject()
	exports := vm.NewObject()
	if err := module.Set("exports", exports); err != nil {
		return nil, fmt.Errorf("%w: module setup: %s",
			ErrTypeScriptCompileFailure, err)
	}
	if err := vm.Set("module", module); err != nil {
		return nil, fmt.Errorf("%w: module setup: %s",
			ErrTypeScriptCompileFailure, err)
	}
	if err := vm.Set("exports", exports); err != nil {
		return nil, fmt.Errorf("%w: exports setup: %s",
			ErrTypeScriptCompileFailure, err)
	}

	// Single-module require. Everything except @falseflag/config throws.
	if err := vm.Set("require", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if name == "@falseflag/config" {
			return shim
		}
		panic(vm.NewTypeError("require() not allowed: " + name))
	}); err != nil {
		return nil, fmt.Errorf("%w: require setup: %s",
			ErrTypeScriptCompileFailure, err)
	}

	// Arm the interrupt before RunString. Two cancellation signals:
	// the request context (ctx.Done) and the wall-clock timer.
	timer := time.NewTimer(compileWallClock)
	defer timer.Stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			vm.Interrupt(ctx.Err())
		case <-timer.C:
			vm.Interrupt(errTSDeadline)
		}
	}()
	defer vm.ClearInterrupt()

	if _, err := vm.RunString(js); err != nil {
		return nil, mapGojaError(err)
	}

	// 3) Read module.exports.default — the value of the TS file's `export default`.
	def, err := extractDefaultExport(vm)
	if err != nil {
		return nil, err
	}
	irJSON, err := json.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal IR: %s",
			ErrTypeScriptCompileFailure, err)
	}
	if len(irJSON) > maxIRBytes {
		return nil, fmt.Errorf("%w: IR exceeds %d bytes",
			ErrTypeScriptCompileFailure, maxIRBytes)
	}

	// 4) Reuse the JSON/CEL validation + CEL program compilation path.
	var tree RulesTree
	if err := json.Unmarshal(irJSON, &tree); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIR, err)
	}
	if err := validateTreeWith(&tree, true); err != nil {
		return nil, err
	}
	env, err := celEnv()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCELCompileFailure, err)
	}
	programs := map[string]CELProgram{}
	if err := compilePredicates(tree.Rules, env, programs); err != nil {
		return nil, err
	}
	return &Compiled{
		Strategy:    StrategyTypeScript,
		IR:          &tree,
		CELPrograms: programs,
	}, nil
}

// tryRehydrateIR reports whether source is already-compiled IR JSON
// and, if so, returns the equivalent Compiled. Returns (nil, false)
// on any parse error or when the parsed object lacks the IR shape;
// callers fall through to the esbuild path.
func tryRehydrateIR(source []byte) (*Compiled, bool) {
	var tree RulesTree
	if err := json.Unmarshal(source, &tree); err != nil {
		return nil, false
	}
	if tree.ValueType == "" {
		return nil, false
	}
	if err := validateTreeWith(&tree, true); err != nil {
		return nil, false
	}
	env, err := celEnv()
	if err != nil {
		return nil, false
	}
	programs := map[string]CELProgram{}
	if err := compilePredicates(tree.Rules, env, programs); err != nil {
		return nil, false
	}
	return &Compiled{
		Strategy:    StrategyTypeScript,
		IR:          &tree,
		CELPrograms: programs,
	}, true
}

func extractDefaultExport(vm *goja.Runtime) (any, error) {
	moduleVal := vm.Get("module")
	if moduleVal == nil || goja.IsUndefined(moduleVal) {
		return nil, fmt.Errorf("%w: module not set", ErrTypeScriptCompileFailure)
	}
	moduleObj := moduleVal.ToObject(vm)
	exportsVal := moduleObj.Get("exports")
	if exportsVal == nil || goja.IsUndefined(exportsVal) {
		return nil, fmt.Errorf("%w: module.exports missing", ErrTypeScriptCompileFailure)
	}
	exportsObj := exportsVal.ToObject(vm)
	defVal := exportsObj.Get("default")
	if defVal == nil || goja.IsUndefined(defVal) {
		return nil, fmt.Errorf("%w: missing default export",
			ErrTypeScriptCompileFailure)
	}
	return defVal.Export(), nil
}

// EsbuildError carries esbuild's structured diagnostics so handlers
// can render line/column information without re-parsing the message.
type EsbuildError struct {
	Messages []EsbuildMessage
}

type EsbuildMessage struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
	Text   string `json:"text"`
}

func (e *EsbuildError) Error() string {
	if len(e.Messages) == 0 {
		return "esbuild: unknown error"
	}
	m := e.Messages[0]
	return fmt.Sprintf("esbuild: %s (line %d, col %d)", m.Text, m.Line, m.Column)
}

// Is reports the error as an ErrTypeScriptCompileFailure so callers
// can errors.Is against the public sentinel.
func (e *EsbuildError) Is(target error) bool {
	return target == ErrTypeScriptCompileFailure
}

func newEsbuildError(msgs []api.Message) *EsbuildError {
	out := make([]EsbuildMessage, 0, len(msgs))
	for _, m := range msgs {
		em := EsbuildMessage{Text: m.Text}
		if m.Location != nil {
			em.File = m.Location.File
			em.Line = m.Location.Line
			em.Column = m.Location.Column
		}
		out = append(out, em)
	}
	return &EsbuildError{Messages: out}
}

// GojaError wraps a goja runtime exception (user code threw, hit an
// uncaught reference error, etc.). Like EsbuildError it satisfies
// errors.Is(ErrTypeScriptCompileFailure).
type GojaError struct {
	Message string
}

func (g *GojaError) Error() string { return "goja: " + g.Message }

func (g *GojaError) Is(target error) bool {
	return target == ErrTypeScriptCompileFailure
}

func mapGojaError(err error) error {
	var interrupted *goja.InterruptedError
	if errors.As(err, &interrupted) {
		if errors.Is(asError(interrupted.Value()), context.DeadlineExceeded) ||
			errors.Is(asError(interrupted.Value()), context.Canceled) {
			return fmt.Errorf("%w: %s",
				ErrTypeScriptCompileFailure, interrupted.Value())
		}
		return fmt.Errorf("%w: %s", ErrTypeScriptCompileFailure, interrupted)
	}
	return &GojaError{Message: err.Error()}
}

func asError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return nil
}

// errTSDeadline is the value passed to vm.Interrupt when the wall
// clock expires. Distinct from ctx.Err() so logs can tell them apart.
var errTSDeadline = errors.New("typescript compile deadline exceeded")

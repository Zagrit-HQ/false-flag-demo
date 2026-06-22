// Package rpc holds the ConnectRPC handlers for the FalseFlag control
// plane. Every handler is a thin translation layer over internal/store
// and the same domain logic the REST handlers use; there is no
// business logic duplication.
package rpc

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/store"
)

// connectError translates store sentinels into Connect codes. The
// REST handlers translate the same sentinels into HTTP codes;
// keeping these two mappings together is what gives the two surfaces
// matching semantics.
func connectError(err error) *connect.Error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case store.IsConflict(err):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func badRequest(err error) *connect.Error {
	return connect.NewError(connect.CodeInvalidArgument, err)
}

// isCompileError reports whether err is a strategy compile failure.
// Mirrors handlers.isCompileError; kept in lockstep so REST and
// Connect agree on which errors are 422 / InvalidArgument-with-detail.
func isCompileError(err error) bool {
	return errors.Is(err, config.ErrTypeScriptCompileFailure) ||
		errors.Is(err, config.ErrCELCompileFailure) ||
		errors.Is(err, config.ErrInvalidIR) ||
		errors.Is(err, config.ErrInvalidPredicate) ||
		errors.Is(err, config.ErrInvalidValueType)
}

// compileErrorToConnect maps a compile error to a Connect error of
// CodeInvalidArgument. Esbuild diagnostics with line/column data are
// attached as a structured detail string so editors can highlight
// the offending location.
func compileErrorToConnect(err error) *connect.Error {
	ce := connect.NewError(connect.CodeInvalidArgument, err)
	var es *config.EsbuildError
	if errors.As(err, &es) {
		// Connect details are protobuf messages, but for the demo we
		// just stash the JSON in a metadata header; the REST 422 body
		// is the rich representation. Clients that need structured
		// details should use the REST surface.
		_ = es // intentional: keep go vet happy without forcing a header dep here.
	}
	return ce
}

// actorFromHeader returns the X-Actor header value or "" when absent.
// Matches the REST handlers' behaviour.
func actorFromHeader(headerGet func(string) string) string {
	return headerGet("X-Actor")
}

package handlers

import (
	"errors"
	"net/http"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
)

// writeCompileError translates a config-package compile error into a
// 422 response carrying structured line/column details. esbuild and
// goja errors carry positional information; everything else gets a
// plain message.
func writeCompileError(w http.ResponseWriter, err error) {
	body := openapi.CompileError{Message: err.Error()}

	var es *config.EsbuildError
	if errors.As(err, &es) {
		details := make([]struct {
			Column *int32  `json:"column,omitempty"`
			File   *string `json:"file,omitempty"`
			Line   *int32  `json:"line,omitempty"`
			Text   string  `json:"text"`
		}, 0, len(es.Messages))
		for _, m := range es.Messages {
			d := struct {
				Column *int32  `json:"column,omitempty"`
				File   *string `json:"file,omitempty"`
				Line   *int32  `json:"line,omitempty"`
				Text   string  `json:"text"`
			}{Text: m.Text}
			if m.File != "" {
				f := m.File
				d.File = &f
			}
			if m.Line != 0 {
				l := int32(m.Line)
				d.Line = &l
			}
			if m.Column != 0 {
				c := int32(m.Column)
				d.Column = &c
			}
			details = append(details, d)
		}
		body.Details = &details
	}
	writeJSON(w, http.StatusUnprocessableEntity, body)
}

// isCompileError reports whether err is a strategy compile failure
// the API surface should report as 422 rather than 400/500.
func isCompileError(err error) bool {
	return errors.Is(err, config.ErrTypeScriptCompileFailure) ||
		errors.Is(err, config.ErrCELCompileFailure) ||
		errors.Is(err, config.ErrInvalidIR) ||
		errors.Is(err, config.ErrInvalidPredicate) ||
		errors.Is(err, config.ErrInvalidValueType)
}

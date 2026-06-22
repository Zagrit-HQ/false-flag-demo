package sdkgo

import (
	"context"

	"github.com/depot/falseflag/internal/eval"
)

// provider is the concrete Provider implementation returned by
// NewProvider.
type provider struct {
	name   string
	client *Client
}

// NewProvider wraps a Client with the OpenFeature-shaped Provider
// interface. If name is "", "falseflag" is used.
func NewProvider(client *Client, name string) Provider {
	if name == "" {
		name = "falseflag"
	}
	return &provider{name: name, client: client}
}

func (p *provider) Metadata() ProviderMetadata {
	return ProviderMetadata{Name: p.name}
}

// typed coerces d.Value into the requested type. If the value matches,
// the decision is returned as-is. If not, the supplied default is
// substituted. The reason is rewritten to type_mismatch unless the
// underlying reason already indicates a fundamental error (no
// snapshot loaded) — in that case the "error" reason is preserved so
// callers can distinguish "wrong type" from "no snapshot available".
func typed[T any](d Decision, def T, ok bool) Decision {
	if ok {
		return d
	}
	if d.Reason != eval.ReasonError {
		d.Reason = eval.ReasonTypeMismatch
	}
	d.Value = def
	return d
}

func (p *provider) BooleanEvaluation(_ context.Context, key string, def bool, evalCtx EvalContext) Decision {
	d := p.client.Evaluate(key, evalCtx)
	v, ok := d.Value.(bool)
	if ok {
		d.Value = v
	}
	return typed(d, def, ok)
}

func (p *provider) StringEvaluation(_ context.Context, key string, def string, evalCtx EvalContext) Decision {
	d := p.client.Evaluate(key, evalCtx)
	v, ok := d.Value.(string)
	if ok {
		d.Value = v
	}
	return typed(d, def, ok)
}

func (p *provider) NumberEvaluation(_ context.Context, key string, def float64, evalCtx EvalContext) Decision {
	d := p.client.Evaluate(key, evalCtx)
	switch v := d.Value.(type) {
	case float64:
		return typed(d, def, true)
	case int:
		d.Value = float64(v)
		return typed(d, def, true)
	case int64:
		d.Value = float64(v)
		return typed(d, def, true)
	default:
		return typed(d, def, false)
	}
}

func (p *provider) ObjectEvaluation(_ context.Context, key string, def any, evalCtx EvalContext) Decision {
	d := p.client.Evaluate(key, evalCtx)
	_, ok := d.Value.(map[string]any)
	return typed(d, def, ok)
}

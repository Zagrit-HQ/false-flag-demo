# Operator API Types

Go definitions for the FalseFlag v1alpha1 CRDs:
`Project`, `Environment`, `Segment`, `RolloutPolicy`, `Flag`,
`FlagBinding`, `FlagSnapshot`.

Every type lives under `v1alpha1/`. `controller-gen` generates
`zz_generated.deepcopy.go` and the CRDs in `deploy/crds/` from the
markers on these types.

Regenerate with `make generate`.

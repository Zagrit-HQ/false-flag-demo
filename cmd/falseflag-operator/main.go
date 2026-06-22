// Command falseflag-operator is the FalseFlag Kubernetes operator.
// It watches the v1alpha1 CRDs and reconciles them through the slice-3
// Connect API. All logic lives in internal/operator; main stays a
// thin entrypoint per the AGENTS.md <50-line rule.
package main

import (
	"os"

	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/operator"
)

func main() {
	os.Exit(buildinfo.WithGracefulShutdown("operator", operator.Run))
}

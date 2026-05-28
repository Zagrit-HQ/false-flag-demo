// Command falseflag-loadgen is the FalseFlag synthetic workload generator.
// Slice 1 is just a stub so the binary builds; real seeding and traffic
// generation arrive with the demo polish slice.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/logging"
)

func main() {
	os.Exit(buildinfo.WithGracefulShutdown("loadgen", run))
}

func run(ctx context.Context) error {
	log := logging.New("loadgen")
	log.Info("falseflag-loadgen not yet implemented; exiting cleanly",
		"version", buildinfo.Version,
		"slice", "1",
	)
	select {
	case <-ctx.Done():
		return nil
	default:
		fmt.Fprintln(os.Stderr, "falseflag-loadgen: stub binary — see demo-polish slice")
		return nil
	}
}

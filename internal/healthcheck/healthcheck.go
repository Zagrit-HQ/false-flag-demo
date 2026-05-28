// Package healthcheck provides a tiny self-probe mode for distroless
// service images. Compose healthchecks cannot use curl or a shell there,
// so the service binary probes its own HTTP endpoint.
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// RunFromArgs handles `healthcheck <url>` and exits the process. It returns
// false when args do not request healthcheck mode.
func RunFromArgs(args []string) bool {
	if len(args) != 3 || args[1] != "healthcheck" {
		return false
	}
	os.Exit(run(args[2]))
	return true
}

func run(url string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %s\n", res.Status)
		return 1
	}
	return 0
}

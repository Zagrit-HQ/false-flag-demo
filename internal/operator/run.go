package operator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/operator/clientapi"
	"github.com/depot/falseflag/internal/operator/controllers"
	falseflagv1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// Scheme is the runtime scheme used by the operator. Exposed so tests
// can reuse the same scheme when constructing envtest clients.
var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(falseflagv1alpha1.AddToScheme(Scheme))
}

// Run is the operator entrypoint. cmd/falseflag-operator wraps this in
// buildinfo.WithGracefulShutdown.
func Run(ctx context.Context) error {
	log := logging.New("operator")

	cfg, err := appconfig.LoadOperator()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	log.Info("starting falseflag-operator",
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
		"metrics_addr", cfg.MetricsAddr,
		"health_addr", cfg.HealthProbeAddr,
		"leader_elect", cfg.LeaderElect,
		"api_base_url", cfg.APIBaseURL,
		"actor", cfg.Actor,
	)

	restCfg, err := config.GetConfig()
	if err != nil {
		// In a demo environment without a kubeconfig, this is fine —
		// log and block until ctx is cancelled so the binary still
		// exits cleanly on SIGINT/SIGTERM.
		log.Warn("no kubernetes config available; operator running in idle mode", "error", err.Error())
		<-ctx.Done()
		return nil
	}

	mgr, err := manager.New(restCfg, manager.Options{
		Scheme: Scheme,
		Metrics: server.Options{
			BindAddress: cfg.MetricsAddr,
		},
		HealthProbeBindAddress: cfg.HealthProbeAddr,
		LeaderElection:         cfg.LeaderElect,
		LeaderElectionID:       "falseflag-operator-leader",
	})
	if err != nil {
		return fmt.Errorf("building manager: %w", err)
	}

	api := clientapi.New(cfg.APIBaseURL, cfg.Actor)
	if err := registerReconcilers(mgr, log, api); err != nil {
		return fmt.Errorf("registering reconcilers: %w", err)
	}

	if err := mgr.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("manager start: %w", err)
	}
	return nil
}

// registerReconcilers wires every controller into the manager. Each
// reconciler shares the same APIClient — the operator targets exactly
// one upstream server.
func registerReconcilers(mgr manager.Manager, log *slog.Logger, api *clientapi.Client) error {
	if err := (&controllers.ProjectReconciler{Log: log, API: api}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("project: %w", err)
	}
	if err := (&controllers.EnvironmentReconciler{Log: log, API: api}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("environment: %w", err)
	}
	if err := (&controllers.SegmentReconciler{Log: log, API: api}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("segment: %w", err)
	}
	if err := (&controllers.RolloutPolicyReconciler{Log: log, API: api}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("rolloutpolicy: %w", err)
	}
	if err := (&controllers.FlagReconciler{Log: log, API: api, Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("flag: %w", err)
	}
	if err := (&controllers.FlagBindingReconciler{Log: log, API: api, Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("flagbinding: %w", err)
	}
	if err := (&controllers.FlagSnapshotReconciler{Log: log, API: api}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("flagsnapshot: %w", err)
	}
	return nil
}

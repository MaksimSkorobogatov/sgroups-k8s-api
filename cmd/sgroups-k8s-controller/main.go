// Command sgroups-k8s-controller mirrors a Tenant onto a same-named Namespace.
package main

import (
	goflag "flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"sgroups.io/sgroups-k8s-api/internal/controllers/tenantnamespace"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// AddToWireScheme (not AddToScheme) — controller-runtime needs one GVK per type.
var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToWireScheme(scheme))
}

const (
	defaultMetricsAddr           = ":8080"
	defaultProbeAddr             = ":8081"
	defaultLeaderElectionID      = "sgroups-tenant-namespace-controller"
	defaultLeaderElectionEnabled = true
)

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		leaderElectionID     string
		leaderElectionNS     string
	)

	klog.InitFlags(nil)
	fs := pflag.NewFlagSet("sgroups-k8s-controller", pflag.ExitOnError)
	fs.AddGoFlagSet(goflag.CommandLine)

	fs.StringVar(&metricsAddr, "metrics-bind-address", defaultMetricsAddr,
		"Address the controller metrics server binds to.")
	fs.StringVar(&probeAddr, "health-probe-bind-address", defaultProbeAddr,
		"Address the controller health/readiness probes bind to.")
	fs.BoolVar(&enableLeaderElection, "leader-elect", defaultLeaderElectionEnabled,
		"Enable leader election for controller manager. Recommended even at replicas=1 for clean rollouts.")
	fs.StringVar(&leaderElectionID, "leader-election-id", defaultLeaderElectionID,
		"Resource name of the lease object used for leader election.")
	fs.StringVar(&leaderElectionNS, "leader-election-namespace", "",
		"Namespace where the leader-election Lease lives. Defaults to the pod namespace via downward API.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(goflag.CommandLine)

	if err := fs.Parse(os.Args[1:]); err != nil {
		klog.Fatalf("parse flags: %v", err)
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        leaderElectionID,
		LeaderElectionNamespace: leaderElectionNS,
	})
	if err != nil {
		klog.Fatalf("unable to start manager: %v", err)
	}

	reconciler := &tenantnamespace.Reconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("tenant-namespace-controller"),
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		klog.Fatalf("unable to set up tenant-namespace controller: %v", err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatalf("unable to set up healthz: %v", err)
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		klog.Fatalf("unable to set up readyz: %v", err)
	}

	klog.Infof("starting tenant-namespace controller manager (leader-election=%v)", enableLeaderElection)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("manager.Start: %v", err)
	}
}

package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"time"

	elasticv1 "github.com/flanksource/elasticsearch-exporter/pkg/api/v1"
	"github.com/flanksource/elasticsearch-exporter/pkg/controllers"
	"github.com/flanksource/elasticsearch-exporter/pkg/metrics"
	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	zaplogfmt "github.com/sykesm/zap-logfmt"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func getClient(url, username, password string) (*elastic.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	c, err := elastic.NewSimpleClient(
		elastic.SetURL(url),
		elastic.SetMaxRetries(10),
		elastic.SetBasicAuth(username, password),
		elastic.SetHttpClient(httpClient),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create elasticsearch client")
	}

	return c, nil
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = elasticv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	yaml.FutureLineWrap()
}

func setupLogger(opts zap.Options) {
	configLog := uzap.NewProductionEncoderConfig()
	configLog.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339Nano))
	}
	logfmtEncoder := zaplogfmt.NewEncoder(configLog)

	logger := zap.New(zap.UseFlagOptions(&opts), zap.Encoder(logfmtEncoder))
	ctrl.SetLogger(logger)
}

func runController(cmd *cobra.Command, args []string) {
	metricsAddr, _ := cmd.Flags().GetString("metrics-addr")
	syncPeriod, _ := cmd.Flags().GetDuration("sync-period")
	enableLeaderElection, _ := cmd.Flags().GetBool("enable-leader-election")

	url, _ := cmd.Flags().GetString("url")
	username, _ := cmd.Flags().GetString("username")
	password := os.Getenv("ELASTIC_PASSWORD")

	elasticClient, err := getClient(url, username, password)
	if err != nil {
		setupLog.Error(err, "failed to get elastic client")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		SyncPeriod:         &syncPeriod,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "ba344e13.flanksource.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	controller := &controllers.ElasticMetricReconciler{
		Log:         ctrl.Log.WithName("controllers").WithName("Template"),
		Elastic:     elasticClient,
		MetricStore: metrics.NewMetricStore(),
		Scheme:      mgr.GetScheme(),
	}

	if err = controller.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Template")
		os.Exit(1)
	}

	// // +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func main() {
	opts := zap.Options{Level: zapcore.DebugLevel}
	// opts.BindFlags(flag.CommandLine)
	// flag.Parse()
	setupLogger(opts)

	var root = &cobra.Command{
		Use:   "elasticsearch-exporter",
		Short: "Run elasticsearch logs exporter",
		Args:  cobra.MinimumNArgs(0),
		Run:   runController,
	}
	root.PersistentFlags().String("metrics-addr", ":8080", "The address the metric endpoint binds to.")
	root.PersistentFlags().Bool("enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	root.PersistentFlags().String("url", "", "ElasticSearch url")
	root.PersistentFlags().String("username", "", "ElasticSearch username")
	root.PersistentFlags().Duration("sync-period", 1*time.Minute, "Sync period")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

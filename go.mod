module github.com/flanksource/logs-exporter

go 1.14

require (
	github.com/flanksource/template-operator v0.1.10
	github.com/go-co-op/gocron v0.6.0
	github.com/go-logr/logr v0.3.0
	github.com/olivere/elastic/v7 v7.0.22
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/cobra v1.1.3
	github.com/sykesm/zap-logfmt v0.0.4
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.2
)

replace (
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)

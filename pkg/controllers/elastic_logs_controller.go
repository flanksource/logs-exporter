/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	elasticv1 "github.com/flanksource/logs-exporter/pkg/api/v1"
	"github.com/flanksource/logs-exporter/pkg/metrics"
	"github.com/flanksource/logs-exporter/pkg/query"
	"github.com/flanksource/template-operator/k8s"
	"github.com/go-logr/logr"
	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	documentsCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "elasticsearch_documents_count",
			Help: "A gauge representing documents count by field",
		},
		[]string{"cluster", "type", "value"},
	)
)

func init() {
	prometheus.MustRegister(documentsCount)
}

// ElasticLogsReconciler reconciles a ElasticLogs object
type ElasticLogsReconciler struct {
	ControllerClient client.Client
	Clientset        *kubernetes.Clientset
	Log              logr.Logger
	MetricStore      *metrics.MetricStore
	Interval         time.Duration
	Scheme           *runtime.Scheme
	Cache            *k8s.SchemaCache
}

// +kubebuilder:rbac:groups="metrics.flanksource.com",resources="elasticlogs",verbs="*"
// +kubebuilder:rbac:groups="",resources="secrets",verbs="get;list"

func (r *ElasticLogsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ElasticLogs", req.NamespacedName)

	log.Info("Started reconciling")

	metric := elasticv1.ElasticLogs{}
	if err := r.ControllerClient.Get(ctx, req.NamespacedName, &metric); err != nil {
		if kerrors.IsNotFound(err) {
			log.Error(err, "elastic metric not found")
			return reconcile.Result{}, nil
		}
		log.Error(err, "failed to get elastic metric")
		return reconcile.Result{}, err
	}

	passwordSecret, err := r.Clientset.CoreV1().Secrets(metric.Spec.Password.Namespace).Get(ctx, metric.Spec.Password.Name, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "failed to find password secret")
		return reconcile.Result{}, err
	}
	password, found := passwordSecret.Data[metric.Spec.Password.Key]
	if !found {
		err := errors.Errorf("failed to find field %s in secret %s/%s", metric.Spec.Password.Key, passwordSecret.Namespace, passwordSecret.Name)
		log.Error(err, "failed to find field password")
		return reconcile.Result{}, err
	}
	elasticClient, err := query.GetClient(metric.Spec.URL, metric.Spec.Username, string(password))
	if err != nil {
		log.Error(err, "failed to create elastic client")
		return reconcile.Result{}, err
	}

	if err := r.Query(elasticClient, metric); err != nil {
		log.Error(err, "error querying elastic")
		return reconcile.Result{}, err
	}

	log.Info("Finished reconciling")

	return ctrl.Result{}, nil
}

func (r *ElasticLogsReconciler) Query(elasticClient *elastic.Client, metric elasticv1.ElasticLogs) error {
	log := r.Log.WithValues("ElasticLogs", types.NamespacedName{Name: metric.Name, Namespace: metric.Namespace})

	for _, tuple := range metric.Spec.Tuples {
		log.Info("Query tuple %s", "name", tuple.MetricName)
		if err := r.queryTuple(elasticClient, metric.Spec.Index, tuple); err != nil {
			log.Error(err, "failed to query tuple", "tuple", tuple)
		}
	}

	return nil
}

func (r *ElasticLogsReconciler) queryTuple(elasticClient *elastic.Client, indexName string, tuple elasticv1.Tuple) error {
	q := query.NewQuery(elasticClient, tuple.Aggregate.Field, r.Interval)

	labels := []string{}
	for k, _ := range tuple.Filters {
		labels = append(labels, k)
	}
	labels = append(labels, aggregateName(tuple.Aggregate.Name))
	gauge := r.MetricStore.GetGauge(tuple.MetricName, labels)

	err := query.AllCombinations(elasticClient, indexName, tuple.Filters, func(fieldValues map[string]query.Filter) {
		filters := map[string]string{}
		logPairs := []interface{}{}
		commonLabelMap := map[string]string{}
		for label, v := range fieldValues {
			filters[v.Field] = v.Value
			commonLabelMap[label] = v.Value
			logPairs = append(logPairs, v.Field)
			logPairs = append(logPairs, v.Value)
		}
		r.Log.Info("Query", logPairs...)
		results, err := q.Query(context.Background(), indexName, filters)
		if err != nil {
			r.Log.Error(err, "failed to query", logPairs...)
		}

		for value, docCount := range results {
			labelMap := map[string]string{}
			for k, v := range commonLabelMap {
				labelMap[k] = v
			}
			labelMap[aggregateName(tuple.Aggregate.Name)] = value

			gauge.Set(labelMap, docCount)
		}
	})

	if err != nil {
		return errors.Wrap(err, "failed to run all combinations")
	}
	return nil
}

func (r *ElasticLogsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerClient = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticv1.ElasticLogs{}).
		Complete(r)
}

func aggregateName(label string) string {
	return label
}

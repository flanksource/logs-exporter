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
	"fmt"
	"time"

	elasticv1 "github.com/flanksource/logs-exporter/pkg/api/v1"
	"github.com/flanksource/logs-exporter/pkg/metrics"
	"github.com/flanksource/logs-exporter/pkg/query"
	"github.com/flanksource/template-operator/k8s"
	"github.com/go-logr/logr"
	"github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// ElasticMetricReconciler reconciles a ElasticMetric object
type ElasticMetricReconciler struct {
	ControllerClient client.Client
	Elastic          *elastic.Client
	Log              logr.Logger
	MetricStore      *metrics.MetricStore
	Scheme           *runtime.Scheme
	Cache            *k8s.SchemaCache
}

// +kubebuilder:rbac:groups="*",resources="*",verbs="*"

func (r *ElasticMetricReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("elasticmetric", req.NamespacedName)

	metric := elasticv1.ElasticMetric{}
	if err := r.ControllerClient.Get(ctx, req.NamespacedName, &metric); err != nil {
		if kerrors.IsNotFound(err) {
			log.Error(err, "elastic metric not found")
			return reconcile.Result{}, nil
		}
		log.Error(err, "failed to get elastic metric")
		return reconcile.Result{}, err
	}

	if err := r.Query(metric); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ElasticMetricReconciler) Query(metric elasticv1.ElasticMetric) error {
	log := r.Log.WithValues("elasticmetric", types.NamespacedName{Name: metric.Name, Namespace: metric.Namespace})
	// index, err := query.LatestIndex(r.Elastic, metric.Spec.Index)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to get latest index")
	// }
	// log.Info("Latest index", "index", index)

	for _, tuple := range metric.Spec.Tuples {
		if err := r.queryTuple(metric.Spec.Index, tuple); err != nil {
			log.Error(err, "failed to query tuple", "tuple", tuple)
		}
	}

	return nil
}

func (r *ElasticMetricReconciler) queryTuple(indexName string, tuple elasticv1.Tuple) error {
	q := query.NewQuery(r.Elastic, tuple.Aggregate.Field, 15*time.Minute)

	labels := []string{}
	for k, _ := range tuple.Filters {
		labels = append(labels, k)
	}
	labels = append(labels, aggregateName(tuple.Aggregate.Name))
	gauge := r.MetricStore.GetGauge(tuple.MetricName, labels)

	err := query.AllCombinations(r.Elastic, indexName, tuple.Filters, func(fieldValues map[string]query.Filter) {
		filters := map[string]string{}
		logPairs := []interface{}{}
		commonLabelMap := map[string]string{}
		for label, v := range fieldValues {
			filters[v.Field] = v.Value
			commonLabelMap[label] = v.Value
			logPairs = append(logPairs, v.Field)
			logPairs = append(logPairs, v.Value)
		}
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

func (r *ElasticMetricReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerClient = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticv1.ElasticMetric{}).
		Complete(r)
}

func aggregateName(label string) string {
	return fmt.Sprintf("aggregate_%s", label)
}

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElasticMetricSpec defines the desired state of ElasticMetric
type ElasticMetricSpec struct {
	Index string `json:"index,omitempty"`
	// IndexPrefix string  `json:"indexPrefix,omitempty"`
	Tuples []Tuple `json:"tuple,omitempty"`
}

type Tuple struct {
	MetricName string            `json:"metricName,omitempty"`
	Filters    map[string]string `json:"filters,omitempty"`
	Aggregate  Pair              `json:"aggregate,omitempty"`
}

type Pair struct {
	Name  string `json:"name,omitempty"`
	Field string `json:"field,omitempty"`
}

// ElasticMetricStatus defines the observed state of Template
type ElasticMetricStatus struct {
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope="Cluster"
// ElasticMetric is the Schema for the elasticmetrics API
type ElasticMetric struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticMetricSpec   `json:"spec,omitempty"`
	Status ElasticMetricStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticMetricList contains a list of ElasticMetric
type ElasticMetricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticMetric `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticMetric{}, &ElasticMetricList{})
}

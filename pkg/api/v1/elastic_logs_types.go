package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElasticLogsSpec defines the desired state of ElasticLogs
type ElasticLogsSpec struct {
	Index    string    `json:"index,omitempty"`
	URL      string    `json:"url,omitempty"`
	Username string    `json:"username,omitempty"`
	Password SecretRef `json:"password,omitempty"`
	Tuples   []Tuple   `json:"tuples,omitempty"`
}

type SecretRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Key       string `json:"key,omitempty"`
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

// ElasticLogsStatus defines the observed state of Template
type ElasticLogsStatus struct {
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope="Cluster"
// ElasticLogs is the Schema for the ElasticLogss API
type ElasticLogs struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticLogsSpec   `json:"spec,omitempty"`
	Status ElasticLogsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticLogsList contains a list of ElasticLogs
type ElasticLogsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticLogs `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticLogs{}, &ElasticLogsList{})
}

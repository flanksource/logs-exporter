package metrics

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Gauge struct {
	gauge *prometheus.GaugeVec
}

type GaugeLabel struct {
	Label string
	Value int64
}

type MetricStore struct {
	gauges map[string]*Gauge
	lock   *sync.Mutex
}

func NewMetricStore() *MetricStore {
	store := &MetricStore{
		gauges: map[string]*Gauge{},
		lock:   &sync.Mutex{},
	}
	return store
}

func (ms *MetricStore) GetGauge(name string, labels []string) *Gauge {
	sort.Strings(labels)
	hasher := md5.New()
	hasher.Write([]byte(strings.Join(labels, "/")))
	hash := hex.EncodeToString(hasher.Sum(nil))

	ms.lock.Lock()
	defer ms.lock.Unlock()

	gauge, found := ms.gauges[hash]
	if found {
		return gauge
	}

	gauge = &Gauge{
		gauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: "A gauge representing documents count by field",
			},
			labels,
		),
	}
	metrics.Registry.MustRegister(gauge.gauge)
	ms.gauges[hash] = gauge
	return gauge
}

func (g *Gauge) Set(labels map[string]string, value int64) {
	g.gauge.With(labels).Set(float64(value))
}

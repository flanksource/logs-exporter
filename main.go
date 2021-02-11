package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

const (
	aggregationName = "origins"
)

var (
	fields = map[string]string{
		"kubernetes.namespace": "namespace",
		"kubernetes.node.name": "node-name",
	}

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

func query(client *elastic.Client, indexPrefix string, clusters []string) {
	index, err := latestIndex(client, indexPrefix)
	if err != nil {
		fmt.Printf("Error getting latest index: %s", err)
		return
	}
	fmt.Println("========================")
	fmt.Printf("Latest index: %s\n", index)

	for _, cluster := range clusters {
		for field, fieldType := range fields {
			query := NewQuery(client, cluster, field, 15*time.Minute)
			fmt.Printf("Query: cluster=%s field=%s\n", cluster, field)
			results, err := query.Query(context.Background(), index)
			if err != nil {
				fmt.Printf("Error query: %v", err)
				return
			}
			for k, v := range results {
				documentsCount.WithLabelValues(cluster, fieldType, k).Set(float64(v))
			}
		}
	}

}

func runServer(cmd *cobra.Command, args []string) {
	url, _ := cmd.Flags().GetString("url")
	username, _ := cmd.Flags().GetString("username")
	password := os.Getenv("ELASTIC_PASSWORD")
	indexPrefix, _ := cmd.Flags().GetString("indexPrefix")
	interval, _ := cmd.Flags().GetDuration("interval")
	clusters, _ := cmd.Flags().GetStringArray("clusters")

	client, err := getClient(url, username, password)
	if err != nil {
		fmt.Printf("Error creating client: %v", err)
		return
	}

	scheduler := gocron.NewScheduler(time.UTC)

	scheduler.Every(interval).Do(func() {
		go func() {
			query(client, indexPrefix, clusters)
		}()
	})

	scheduler.StartAsync()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

func main() {
	var root = &cobra.Command{
		Use:   "elasticsearch-exporter",
		Short: "Run elasticsearch logs exporter",
		Args:  cobra.MinimumNArgs(0),
		Run:   runServer,
	}
	root.PersistentFlags().String("indexPrefix", "", "Filebeat index prefix, example: filebeat-7.10.2-")
	root.PersistentFlags().String("url", "", "ElasticSearch url")
	root.PersistentFlags().String("username", "", "ElasticSearch username")
	root.PersistentFlags().Duration("interval", 1*time.Minute, "Query interval")
	root.PersistentFlags().StringArray("clusters", []string{}, "List of clusters")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

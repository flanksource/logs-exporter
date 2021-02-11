package main

import (
	"context"
	"encoding/json"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

type Query struct {
	client      *elastic.Client
	fieldName   string
	clusterName string
	interval    time.Duration
}

type QueryResult map[string]int64

func NewQuery(client *elastic.Client, clusterName, fieldName string, interval time.Duration) *Query {
	query := &Query{
		client:      client,
		fieldName:   fieldName,
		clusterName: clusterName,
		interval:    interval,
	}

	return query
}

func (q *Query) Query(ctx context.Context, indexName string) (QueryResult, error) {
	query := q.getQuery()

	result, err := q.getResult(ctx, indexName, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get result")
	}

	return q.decodeResult(result)
}

func (q *Query) getQuery() elastic.Query {
	now := time.Now()
	formatForES := "2006-01-02T15:04:05-07:00"
	nowStr := now.Format(formatForES)
	ltStr := nowStr
	lt, _ := time.Parse(formatForES, ltStr)
	gt := lt.Add(time.Duration(-1 * q.interval))
	gtStr := gt.Format(formatForES)

	boolQuery := elastic.NewBoolQuery()
	boolQuery.Must(
		elastic.NewTermQuery("fields.cluster", q.clusterName),
		elastic.NewRangeQuery("@timestamp").
			Gt(gtStr).
			Lt(ltStr),
	)

	return boolQuery
}

func (q *Query) getResult(ctx context.Context, indexName string, query elastic.Query) (*elastic.SearchResult, error) {
	aggr := elastic.NewTermsAggregation().Field(q.fieldName).Size(100)
	return q.client.Search().
		Index(indexName).
		Query(query).
		Size(0).
		Aggregation(aggregationName, aggr).
		Pretty(true).
		Do(context.Background())
}

func (q *Query) decodeResult(result *elastic.SearchResult) (QueryResult, error) {
	rawMsg := result.Aggregations[aggregationName]
	var ar elastic.AggregationBucketKeyItems
	err := json.Unmarshal(rawMsg, &ar)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal result")
	}

	qr := QueryResult{}

	for _, item := range ar.Buckets {
		keyStr, ok := item.Key.(string)
		if !ok {
			return nil, errors.Errorf("failed to convert key %v to string", item.Key)
		}
		qr[keyStr] = item.DocCount
	}

	return qr, nil
}

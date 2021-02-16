package query

import (
	"context"
	"encoding/json"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

type Query struct {
	client          *elastic.Client
	fieldName       string
	interval        time.Duration
	aggregationName string
}

type QueryResult map[string]int64

func NewQuery(client *elastic.Client, fieldName string, interval time.Duration) *Query {
	query := &Query{
		client:          client,
		fieldName:       fieldName,
		interval:        interval,
		aggregationName: "documents",
	}

	return query
}

func (q *Query) Query(ctx context.Context, indexName string, fields map[string]string) (QueryResult, error) {
	query := q.getQuery(fields)

	result, err := q.getResult(ctx, indexName, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get result")
	}

	return q.decodeResult(result)
}

func (q *Query) getQuery(fields map[string]string) elastic.Query {
	now := time.Now()
	formatForES := "2006-01-02T15:04:05-07:00"
	nowStr := now.Format(formatForES)
	ltStr := nowStr
	lt, _ := time.Parse(formatForES, ltStr)
	gt := lt.Add(time.Duration(-1 * q.interval))
	gtStr := gt.Format(formatForES)

	queries := []elastic.Query{
		elastic.NewRangeQuery("@timestamp").
			Gt(gtStr).
			Lt(ltStr),
	}

	for k, v := range fields {
		queries = append(queries, elastic.NewTermQuery(k, v))
	}

	boolQuery := elastic.NewBoolQuery()
	boolQuery.Must(queries...)

	return boolQuery
}

func (q *Query) getResult(ctx context.Context, indexName string, query elastic.Query) (*elastic.SearchResult, error) {
	aggr := elastic.NewTermsAggregation().Field(q.fieldName).Size(100)
	return q.client.Search().
		Index(indexName).
		Query(query).
		Size(0).
		Aggregation(q.aggregationName, aggr).
		Pretty(true).
		Do(context.Background())
}

func (q *Query) decodeResult(result *elastic.SearchResult) (QueryResult, error) {
	rawMsg := result.Aggregations[q.aggregationName]
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

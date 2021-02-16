package query

import (
	"context"
	"fmt"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

type Callback func(fieldValues map[string]Filter)

type FieldValues struct {
	Label  string
	Field  string
	Values []string
}

type Filter struct {
	Field string
	Value string
}

func AllCombinations(client *elastic.Client, index string, fieldsMap map[string]string, callback Callback) error {
	allValues := []FieldValues{}

	if len(fieldsMap) == 0 {
		return nil
	}

	for label, field := range fieldsMap {
		values, err := getFieldValues(client, index, field)
		if err != nil {
			return errors.Wrapf(err, "failed to find field values for field=%s label=%s", field, label)
		}
		fmt.Printf("label=%s field=%s values=%v\n", label, field, values)
		allValues = append(allValues, FieldValues{Label: label, Field: field, Values: values})
	}

	getCombinations(allValues, callback, 0, map[string]Filter{})

	return nil
}

func getCombinations(allValues []FieldValues, callback Callback, index int, filters map[string]Filter) {
	if index == len(allValues) {
		callback(filters)
		return
	}

	fieldValue := allValues[index]

	for i := 0; i < len(fieldValue.Values); i++ {
		filters[fieldValue.Label] = Filter{Field: fieldValue.Field, Value: fieldValue.Values[i]}
		getCombinations(allValues, callback, index+1, filters)
	}
}

func getFieldValues(client *elastic.Client, index, field string) ([]string, error) {
	query := NewQuery(client, field, 15*time.Minute)
	results, err := query.Query(context.Background(), index, map[string]string{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get field values")
	}

	values := []string{}
	for k, _ := range results {
		values = append(values, k)
	}
	return values, nil
}

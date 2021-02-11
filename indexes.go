package main

import (
	"context"
	"sort"
	"strings"
	"time"

	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

func latestIndex(client *elastic.Client, indexPrefix string) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	resp, err := client.CatIndices().Do(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to list indexes")
	}

	indexes := []string{}

	for _, index := range resp {
		if strings.HasPrefix(index.Index, indexPrefix) {
			indexes = append(indexes, index.Index)
		}
	}

	sort.Strings(indexes)

	if len(indexes) == 0 {
		return "", errors.Errorf("No index found for index prefix: %s", indexPrefix)
	}

	return indexes[len(indexes)-1], nil
}

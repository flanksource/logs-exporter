package query

import (
	"crypto/tls"
	"net/http"

	elastic "github.com/olivere/elastic/v7"
	"github.com/pkg/errors"
)

func GetClient(url, username, password string) (*elastic.Client, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url),
		elastic.SetMaxRetries(10),
		elastic.SetBasicAuth(username, password),
		elastic.SetHttpClient(httpClient),
	}

	c, err := elastic.NewSimpleClient(options...)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create elasticsearch client")
	}

	return c, nil
}

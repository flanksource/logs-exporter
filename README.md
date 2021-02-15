# logs-exporter

### Usage

```bash
export ELASTIC_PASSWORD=abcdefgh123456789
logs-exporter --url=https://logs.es-cluster.k8s  --username=elastic --indexPrefix=filebeat-7.10.2- --clusters=cluster1-infra --clusters cluster2-infra
```
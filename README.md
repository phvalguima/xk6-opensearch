# xk6 Extension for OpenSearch

This extension builds on top of k6 and provides APIs to interact with OpenSearch.

## Build

```
sudo snap install go --channel=1.20/stable
sudo snap install k6

# Clone this repo and go to the main folder.

xk6 build --with k6/x/opensearch=.
```

## Run

```
./k6 run --vus=10 --duration=10s  examples/script.js
```

## Integrate with Prometheus

Use the native remote-writer available:
```
K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write \
  ./k6 run -o experimental-prometheus-rw --vus=10 --duration=10s examples/script.js
```
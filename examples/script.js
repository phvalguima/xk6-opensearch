import opensearch from 'k6/x/opensearch';
import { Trend, Counter } from 'k6/metrics';

const OpenSearchOperation = {
	Create: 0,
	Delete: 1,
	Search: 2,
	Update: 3,
	Index: 4,
};

const xk6_opensearch_success_http_latency = new Trend('xk6_opensearch_success_http_latency', true);
const xk6_opensearch_success_failure_latency = new Trend('xk6_opensearch_success_failure_latency', true);

const xk6_opensearch_total_latency = new Counter('xk6_opensearch_total_latency');
const xk6_opensearch_total_bytes_sent = new Counter('xk6_opensearch_total_bytes_sent');
const xk6_opensearch_total_bytes_received = new Counter('xk6_opensearch_total_bytes_received');


const client = opensearch.open('admin', 'admin', 'https://localhost:9200');


export function setup() {
    const res = opensearch.index(client, OpenSearchOperation.Create,  'test', 1, 0);
    // console.log(res);
}
  
export function teardown() {
    const res = opensearch.index(client, OpenSearchOperation.Delete, 'test');
}

export default function () {
    // const client = opensearch.open('admin', 'admin', 'https://localhost:9200');
    const res = opensearch.document(client, OpenSearchOperation.Create, 'test', '');
    // Seems that k6 js uses ms as standard unit
    if (res.respstatus < 200 || res.respstatus >= 300) {
        xk6_opensearch_success_failure_latency.add(res.latency / 1e6);
    } else {
        xk6_opensearch_success_http_latency.add(res.latency / 1e6);
    }
    xk6_opensearch_total_latency.add(res.latency / 1e6);
    xk6_opensearch_total_bytes_sent.add(res.bytes_sent);
    xk6_opensearch_total_bytes_received.add(res.bytes_received);
}
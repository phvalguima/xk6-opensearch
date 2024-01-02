import opensearch from 'k6/x/opensearch';
import { Trend } from 'k6/metrics';

const myTrend = new Trend('waiting_time');

const client = opensearch.open('admin', 'admin', 'https://localhost:9200');


export function setup() {
    const res = opensearch.create_index(client, 'test');
}
  
export function teardown() {
    db.close();
}

export default function () {
    // const client = opensearch.open('admin', 'admin', 'https://localhost:9200');
    // const res = opensearch.bulk(client, '{ "index" : { "_index" : "test", "_id" : "1" } }{ "field1" : "value1" }');
    // console.log(res);

}
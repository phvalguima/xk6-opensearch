package opensearch

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	osv2 "github.com/opensearch-project/opensearch-go/v2"
	osapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/opensearch", new(RootModule))
}

type RootModule struct{}

// Instantiated for each new VU
type OpenSearch struct {
	vu     modules.VU
	client *osv2.Client
}

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &OpenSearch{}
)

// NewModuleInstance implements the modules.Module interface to return
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &OpenSearch{vu: vu}
}

func (opensearch *OpenSearch) Exports() modules.Exports {
	return modules.Exports{Default: opensearch}
}

func (*OpenSearch) Open(username string, password string, url string) (*osv2.Client, error) {
	client, err := osv2.NewClient(osv2.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{url},
		Username:  username, // For testing only. Don't store credentials in code.
		Password:  password,
	})

	if err != nil {
		return nil, err
	}
	return client, nil
}

type ConnStats struct {
	Latency  float64
	BytesIn  int
	BytesOut int
}

type OpenSearchRequest struct {
	op   int    `json:"op"`
	data string `json:"data"`
}
type KVResponse map[string]interface{}

// Using the Do method, as opensearch-go implements its own logic directly on top of
// Golang's http.Client.Do: https://github.com/opensearch-project/opensearch-go/blob/f5e372a97b740bc7d12fb46759945a567b3cee55/opensearch.go#L272
// Then, collect the statistics needed here
func (*OpenSearch) do(client *osv2.Client, ctx context.Context, req Request, dataPointer interface{}) (*ConnStats, *Response, error) {
	start := time.Now().Nanoseconds()

	resp, err := osv2.Client.Do(ctx, req, dataPointer)

	var stats *ConnStats = &ConnStats{}
	stats.Latency = time.Since(start).Nanoseconds()
	stats.BytesIn = len(req.Body)
	stats.BytesOut = len(resp.Body)

	return stats, resp, nil
}

func (os *OpenSearch) CreateIndex(client *osv2.Client, doc string, indexName string, docId string) (interface{}, error) {
	req, err := osapi.DocumentDeleteReq{
		Index:      indexName,
		DocumentID: docId,
		Body: strings.NewReader(doc)
	}.GetRequest()
	if err != nil {
		return nil, err
	}
	resp, err := os.do(client, context.Background(), req, osapi.DocumentCreateResp{})
	return resp, nil
}

func (os *OpenSearch) CreateDocument(client *osv2.Client, doc string, indexName string, docId string) (interface{}, error) {
	req, err := osapi.DocumentDeleteReq{
		Index:      indexName,
		DocumentID: docId,
		Body: strings.NewReader(doc)
	}.GetRequest()
	if err != nil {
		return nil, err
	}
	resp, err := os.do(client, context.Background(), req, osapi.DocumentCreateResp{})
	return resp, nil
}

func (os *OpenSearch) DeleteDocument(client *osv2.Client, indexName string, docId string) (interface{}, error) {
	req, err := osapi.DocumentDeleteReq{
		Index:      indexName,
		DocumentID: docId,
	}.GetRequest()
	if err != nil {
		return nil, err
	}
	resp, err := os.do(client, context.Background(), req, osapi.DocumentDeleteResp{})
	return resp, nil
}

func (*OpenSearch) oldBulk(client *osv2.Client, bulkdata string) (interface{}, error) {

	blk, err := client.Bulk(
		strings.NewReader(bulkdata),
	)
	if err != nil {
		return "", err
	}
	/*
		var query []KVRequest
		err := json.Unmarshal([]byte(bulkdata), &query)
		if err != nil {
			return nil, err
		}
		blk, err := client.Bulk(
			strings.NewReader(strings.Join(query, "\n")),
		)
	*/
	return blk, nil
	/*
		    req := opensearchapi.IndexRequest{
		        Index:      IndexName,
		        Body:       document,
		    }
			if docId != nil {
				req.DocumentID = docId;
			}
		    insertResponse, err := req.Do(context.Background(), client)
			if err != nil {
				return nil, err
			}
			return "success", nil
	*/
}

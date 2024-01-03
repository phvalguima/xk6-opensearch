package opensearch

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	osv3 "github.com/opensearch-project/opensearch-go/v3"
	osapi "github.com/opensearch-project/opensearch-go/v3/opensearchapi"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/opensearch", new(RootModule))
}

type RootModule struct{}

// Instantiated for each new VU
type OpenSearch struct {
	vu     modules.VU
	client *osv3.Client
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

func (*OpenSearch) Open(username string, password string, url string) (*osv3.Client, error) {
	client, err := osv3.NewClient(osv3.Config{
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
	Latency       int64
	BytesSent     int
	BytesReceived int
	RespStatus    int
}

const (
	Create = iota // value = 0
	Delete
	Search
	Update
	Index
)

// Use this to generate random strings
// TODO: move this entire logic to a separated goroutine, managed at the root module level. We should just read a channel with latest random strings
const runes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(n int) string {

	// Make the seed configurable, so we can repeat experiments
	rand.Seed(time.Now().UnixNano())

	vec := make([]byte, n)
	for i := range vec {
		vec[i] = runes[rand.Intn(len(runes))]
	}
	return string(vec)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Using the Do method, as opensearch-go implements its own logic directly on top of
// Golang's http.Client.Do: https://github.com/opensearch-project/opensearch-go/blob/f5e372a97b740bc7d12fb46759945a567b3cee55/opensearch.go#L272
// Then, collect the statistics needed here
func (*OpenSearch) do(client *osv3.Client, ctx context.Context, req osv3.Request, dataPointer interface{}) (*ConnStats, error) {
	start := time.Now().UnixNano()

	resp, err := client.Do(ctx, req, nil)

	if err != nil {
		return nil, err
	}

	var stats *ConnStats = &ConnStats{}
	stats.Latency = time.Now().UnixNano() - start
	stats.BytesSent = 0
	stats.BytesReceived = 0
	http, err := req.GetRequest()
	if err != nil {
		return nil, err
	}
	if http.Body != nil {
		snd, err := io.ReadAll(http.Body)
		if err != nil {
			return nil, err
		}
		stats.BytesSent = len(snd)
	}
	if resp.Body != nil {
		rcv, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		stats.BytesReceived = len(rcv)
	}
	stats.RespStatus = resp.StatusCode
	return stats, nil
}

func (os *OpenSearch) Index(client *osv3.Client, op int, indexName string, number_of_shards int, number_of_replicas int) (interface{}, error) {

	var datapointer interface{}
	var req osv3.Request
	switch op {
	case Create:
		req = osapi.IndicesCreateReq{
			Index: indexName,
			Body: strings.NewReader(`{
					"settings": {
						"index": {
							"number_of_shards": ` + strconv.Itoa(number_of_shards) + `,
							"number_of_replicas": ` + strconv.Itoa(number_of_replicas) + `
						}
					}
				}`),
		}
		datapointer = osapi.IndicesCreateResp{}
	case Delete:
		req = osapi.IndicesDeleteReq{
			Indices: []string{indexName},
		}
		datapointer = osapi.IndicesDeleteResp{}
	default:
		return nil, errors.New("Invalid operation")
	}

	stats, err := os.do(client, context.Background(), req, datapointer)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (os *OpenSearch) Document(client *osv3.Client, op int, indexName string, docId string, data ...string) (interface{}, error) {

	var datapointer interface{}
	var req osv3.Request
	var body, id string
	switch op {
	case Create:
		if len(data) > 0 {
			body = data[0]
		} else {
			body = GenerateRandomString(100)
		}
		if len(docId) > 0 {
			id = docId
		} else {
			id = "id-" + GenerateRandomString(6)
		}
		req = osapi.DocumentCreateReq{
			Index:      indexName,
			DocumentID: id,
			Body: strings.NewReader(`{
					"data": "` + body + `"
			}`),
		}
		datapointer = osapi.DocumentCreateResp{}
	case Delete:
		req = osapi.DocumentDeleteReq{
			Index:      indexName,
			DocumentID: docId,
		}
		datapointer = osapi.DocumentDeleteResp{}
	default:
		return nil, errors.New("Invalid operation")
	}

	stats, err := os.do(client, context.Background(), req, datapointer)
	if err != nil {
		return nil, err
	}
	// Slight correction, as io.ReadCloser is only read once and we do it at Do() call
	switch op {
	case Create:
		stats.BytesSent = len(body)
	}
	return stats, nil
}

/*
func (os *OpenSearch) CreateDocument(client *osv3.Client, doc string, indexName string, docId string) (interface{}, error) {
	req, err := osapi.DocumentCreateReq{
		Index:      indexName,
		DocumentID: docId,
		Body:       strings.NewReader(doc),
	}
	if err != nil {
		return nil, err
	}
	stats, resp, err := os.do(client, context.Background(), req, osapi.DocumentCreateResp{})
	return stats, resp, nil
}

func (os *OpenSearch) DeleteDocument(client *osv3.Client, indexName string, docId string) (interface{}, error) {
	req, err := osapi.DocumentDeleteReq{
		Index:      indexName,
		DocumentID: docId,
	}
	if err != nil {
		return nil, err
	}
	stats, resp, err := os.do(client, context.Background(), req, osapi.DocumentDeleteResp{})
	return stats, resp, nil
}
*/

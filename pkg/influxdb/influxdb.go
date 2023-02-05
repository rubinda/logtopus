package influxdb

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const (
	// defaultQueryRangeStart represents lowest possible time value (InfluxDB) and is used when nothing provided.
	defaultQueryRangeStart = 0
)

// Client contains methods for database interaction.
type Client struct {
	// influxClient is a client to connect to InfluxDB.
	influxClient influxdb2.Client
	// defaultWriteApi contains the write operations to the database.
	defaultWriteApi api.WriteAPI
	// Org is the organization identifier for storing data.
	Org string
	// Bucket is the bucket name for storing data.
	Bucket string
}

// Configuration represents database parameters.
type Configuration struct {
	// ServerURL contains the URL and port of InfluxDB.
	ServerURL string
	// Token is an authentication token for InfluxDB.
	Token string
	// InfluxOrg is the organization identifier for storing data.
	InfluxOrg string
	// InfluxBucket is the bucket name for storing data.
	InfluxBucket string
}

// NewClient initiates a new connection to InfluxDB based on given configuration.
func NewClient(c Configuration) *Client {
	// Provide a single HTTP client that can be reused. According to documentation it should be thread safe.
	httpClient := &http.Client{
		Timeout: time.Second * time.Duration(60),
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	influxClient := influxdb2.NewClientWithOptions(c.ServerURL, c.Token,
		influxdb2.DefaultOptions().SetHTTPClient(httpClient).SetPrecision(time.Millisecond),
	)
	// TODO:
	//  - one could use a non-blocking writeApi,error isn't returned to client - bad for testing purposes
	// defaultWriteApi := influxClient.WriteAPI(c.InfluxOrg, c.InfluxBucket)
	// errCh := defaultWriteApi.Errors()
	// go func() {
	// 	// TODO:
	// 	//  - triggering a notification to a system administrator would be a great addition
	// 	for err := range errCh {
	// 		log.Printf("[WARNING] InfluxDB write error: %s \n", err.Error())
	// 	}
	// }()
	return &Client{influxClient, nil, c.InfluxOrg, c.InfluxBucket}
}

// StoreEvent writes event data to the database.
func (c *Client) StoreEvent(eventData BasicEvent) error {
	writeApi := c.influxClient.WriteAPIBlocking(c.Org, c.Bucket)
	influxPoint, err := eventData.ToPoint()
	if err != nil {
		return err
	}
	return writeApi.WritePoint(context.Background(), influxPoint)
}

// QueryEvents runs a query, where queryFields are fields in InfluxDB. Returns results grouped (pivoted) by timestamp.
func (c *Client) QueryEvents(queryFields map[string]any) ([]BasicEvent, error) {
	queryApi := c.influxClient.QueryAPI(c.Org)
	// TODO:
	//  - QueryWithParams is currently only supported for InfluxDB Cloud and doesn't support this usecase anyway :(
	result, err := queryApi.Query(context.Background(), queryBuilder(queryFields, c.Bucket))
	if err != nil {
		return nil, err
	}
	return QueryResultsToBasicEvents(result)
}

// Disconnect (gracefully) shuts down the connection to InfluxDB if it is active.
func (c *Client) Disconnect() {
	if c.influxClient != nil {
		c.influxClient.Close()
	}
}

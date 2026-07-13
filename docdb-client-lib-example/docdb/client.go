package docdb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"

	docdbv1 "github.com/docdb/client/gen/docdb/v1"
	"github.com/docdb/client/gen/docdb/v1/docdbv1connect"
)

// Client is the DocDB client. Create one with NewClient and reuse it across
// requests; it is safe for concurrent use.
type Client struct {
	rpc     docdbv1connect.DocDBClient
	baseURL string
}

// Option configures a Client.
type Option func(*clientConfig)

type clientConfig struct {
	httpClient connect.HTTPClient
	opts       []connect.ClientOption
}

// WithHTTPClient sets a custom HTTP client (e.g. with TLS or timeouts).
func WithHTTPClient(hc connect.HTTPClient) Option {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithGRPC forces the gRPC wire protocol (binary, HTTP/2 only).
// Use this when connecting to a standard gRPC server (non-Connect).
func WithGRPC() Option {
	return func(c *clientConfig) {
		c.opts = append(c.opts, connect.WithGRPC())
	}
}

// NewClient creates a Client that connects to baseURL (e.g. "http://localhost:60013").
// By default it uses an h2c (HTTP/2 cleartext) transport, which speaks both
// Connect and gRPC wire protocols without TLS.
func NewClient(baseURL string, opts ...Option) *Client {
	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					// h2c: plain TCP — no TLS.
					return (&net.Dialer{}).DialContext(ctx, network, addr)
				},
			},
		}
	}

	rpc := docdbv1connect.NewDocDBClient(cfg.httpClient, baseURL, cfg.opts...)
	return &Client{rpc: rpc, baseURL: baseURL}
}

// Close is a no-op for HTTP clients but is included for compatibility.
func (c *Client) Close() {}

// ── Ping ──

type PingInfo struct {
	Version   string
	GoVersion string
	DataDir   string
}

// Ping checks that the server is alive and returns version info.
func (c *Client) Ping(ctx context.Context) (*PingInfo, error) {
	resp, err := c.rpc.Ping(ctx, connect.NewRequest(&docdbv1.PingRequest{}))
	if err != nil {
		return nil, err
	}
	return &PingInfo{
		Version:   resp.Msg.Version,
		GoVersion: resp.Msg.GoVersion,
		DataDir:   resp.Msg.DataDir,
	}, nil
}

// ── Raw commands ──

// Execute runs any DocDB command string and returns the result.
func (c *Client) Execute(ctx context.Context, command string) (*Result, error) {
	resp, err := c.rpc.Execute(ctx, connect.NewRequest(&docdbv1.ExecuteRequest{Command: command}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if !msg.Ok {
		return nil, fmt.Errorf("docdb: %s", msg.Error)
	}
	result := &Result{Message: msg.Message}
	if len(msg.Docs) > 0 {
		result.Docs = decodeDocuments(msg.Docs)
	}
	return result, nil
}

// Query runs a find command via streaming RPC and returns all docs.
func (c *Client) Query(ctx context.Context, command string) (*Result, error) {
	stream, err := c.rpc.Query(ctx, connect.NewRequest(&docdbv1.QueryRequest{Command: command}))
	if err != nil {
		return nil, err
	}
	result := &Result{}
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Error != "" {
			return nil, fmt.Errorf("docdb: %s", msg.Error)
		}
		if msg.Doc != nil {
			result.Docs = append(result.Docs, decodeDocument(msg.Doc))
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// QueryStream runs a find command and calls fn for each document as it arrives.
func (c *Client) QueryStream(ctx context.Context, command string, fn func(doc Doc) error) error {
	stream, err := c.rpc.Query(ctx, connect.NewRequest(&docdbv1.QueryRequest{Command: command}))
	if err != nil {
		return err
	}
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Error != "" {
			return fmt.Errorf("docdb: %s", msg.Error)
		}
		if msg.Doc != nil {
			if err := fn(decodeDocument(msg.Doc)); err != nil {
				return err
			}
		}
	}
	return stream.Err()
}

// ── Schema/Collection operations ──

// ListCollections returns all collection names on the server.
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	resp, err := c.rpc.ListCollections(ctx, connect.NewRequest(&docdbv1.ListCollectionsRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.Collections, nil
}

// DescribeCollection returns metadata of a single collection.
func (c *Client) DescribeCollection(ctx context.Context, name string) (*CollectionInfo, error) {
	resp, err := c.rpc.DescribeCollection(ctx, connect.NewRequest(&docdbv1.DescribeCollectionRequest{Collection: name}))
	if err != nil {
		return nil, err
	}
	return &CollectionInfo{
		Name:     resp.Msg.Collection,
		DocCount: resp.Msg.DocCount,
		Size:     resp.Msg.SizeBytes,
	}, nil
}

// CreateCollection creates a collection.
func (c *Client) CreateCollection(ctx context.Context, name string) error {
	cmd := fmt.Sprintf("db.createCollection(%q)", name)
	_, err := c.Execute(ctx, cmd)
	return err
}

// DropCollection drops a collection.
func (c *Client) DropCollection(ctx context.Context, name string) error {
	cmd := fmt.Sprintf("db.dropCollection(%q)", name)
	_, err := c.Execute(ctx, cmd)
	return err
}

// Collection returns a CollectionQuery builder for the collection.
func (c *Client) Collection(name string) *CollectionQuery {
	return &CollectionQuery{client: c, collectionName: name}
}

// ── Decoding helpers ──

func decodeDocuments(pbDocs []*docdbv1.Document) []Doc {
	docs := make([]Doc, len(pbDocs))
	for i, pbDoc := range pbDocs {
		docs[i] = decodeDocument(pbDoc)
	}
	return docs
}

func decodeDocument(pbDoc *docdbv1.Document) Doc {
	doc := make(Doc, len(pbDoc.Fields))
	for k, v := range pbDoc.Fields {
		doc[k] = decodeValue(v)
	}
	return doc
}

func decodeValue(v *docdbv1.Value) any {
	if v == nil || v.IsNull {
		return nil
	}
	switch k := v.Kind.(type) {
	case *docdbv1.Value_IntVal:
		return k.IntVal
	case *docdbv1.Value_FloatVal:
		return k.FloatVal
	case *docdbv1.Value_TextVal:
		return k.TextVal
	case *docdbv1.Value_BoolVal:
		return k.BoolVal
	}
	return nil
}

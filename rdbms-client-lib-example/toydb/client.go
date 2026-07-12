package toydb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"

	toydbv1 "github.com/toydb/client/gen/toydb/v1"
	"github.com/toydb/client/gen/toydb/v1/toydbv1connect"
)

// Client is the ToyDB client. Create one with NewClient and reuse it across
// requests; it is safe for concurrent use.
type Client struct {
	rpc     toydbv1connect.ToyDBClient
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

// NewClient creates a Client that connects to baseURL (e.g. "http://localhost:9090").
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

	rpc := toydbv1connect.NewToyDBClient(cfg.httpClient, baseURL, cfg.opts...)
	return &Client{rpc: rpc, baseURL: baseURL}
}

// Close is a no-op for HTTP clients but is included for future use.
func (c *Client) Close() {}

// ── Ping ──────────────────────────────────────────────────────────────────────

// PingInfo is returned by Ping.
type PingInfo struct {
	Version   string
	GoVersion string
	DataDir   string
}

// Ping checks that the server is alive and returns version info.
func (c *Client) Ping(ctx context.Context) (*PingInfo, error) {
	resp, err := c.rpc.Ping(ctx, connect.NewRequest(&toydbv1.PingRequest{}))
	if err != nil {
		return nil, err
	}
	return &PingInfo{
		Version:   resp.Msg.Version,
		GoVersion: resp.Msg.GoVersion,
		DataDir:   resp.Msg.DataDir,
	}, nil
}

// ── Raw SQL ───────────────────────────────────────────────────────────────────

// Execute runs any SQL statement and returns the result.
// For SELECT statements the Result.Rows field is populated.
// For DDL/DML the Result.Message field is populated.
func (c *Client) Execute(ctx context.Context, sql string) (*Result, error) {
	resp, err := c.rpc.Execute(ctx, connect.NewRequest(&toydbv1.ExecuteRequest{Sql: sql}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if !msg.Ok {
		return nil, fmt.Errorf("toydb: %s", msg.Error)
	}
	result := &Result{Message: msg.Message}
	if msg.Result != nil {
		result.Columns = msg.Result.Columns
		result.Rows = decodeResultSet(msg.Result)
	}
	return result, nil
}

// Query runs a SELECT via the streaming RPC and returns all rows.
// For very large tables prefer QueryStream to avoid buffering the whole result.
func (c *Client) Query(ctx context.Context, sql string) (*Result, error) {
	stream, err := c.rpc.Query(ctx, connect.NewRequest(&toydbv1.QueryRequest{Sql: sql}))
	if err != nil {
		return nil, err
	}
	result := &Result{}
	first := true
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Error != "" {
			return nil, fmt.Errorf("toydb: %s", msg.Error)
		}
		if first && len(msg.Columns) > 0 {
			result.Columns = msg.Columns
			first = false
		}
		if msg.Row != nil {
			result.Rows = append(result.Rows, decodeRow(msg.Row, result.Columns))
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// QueryStream runs a SELECT and calls fn for each row as it arrives.
// The fn receives the column names on the first call (via the Result.Columns
// slice) and one row per subsequent call.
func (c *Client) QueryStream(ctx context.Context, sql string, fn func(columns []string, row Row) error) error {
	stream, err := c.rpc.Query(ctx, connect.NewRequest(&toydbv1.QueryRequest{Sql: sql}))
	if err != nil {
		return err
	}
	var cols []string
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Error != "" {
			return fmt.Errorf("toydb: %s", msg.Error)
		}
		if len(msg.Columns) > 0 {
			cols = msg.Columns
		}
		if msg.Row != nil {
			row := decodeRow(msg.Row, cols)
			if err := fn(cols, row); err != nil {
				return err
			}
		}
	}
	return stream.Err()
}

// ── Schema operations ─────────────────────────────────────────────────────────

// ListTables returns all table names on the server.
func (c *Client) ListTables(ctx context.Context) ([]string, error) {
	resp, err := c.rpc.ListTables(ctx, connect.NewRequest(&toydbv1.ListTablesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.Tables, nil
}

// DescribeTable returns the schema of the named table.
func (c *Client) DescribeTable(ctx context.Context, table string) (*TableSchemaInfo, error) {
	resp, err := c.rpc.DescribeTable(ctx, connect.NewRequest(&toydbv1.DescribeTableRequest{Table: table}))
	if err != nil {
		return nil, err
	}
	cols := make([]Column, len(resp.Msg.Columns))
	for i, cd := range resp.Msg.Columns {
		cols[i] = Column{Name: cd.Name, Type: ColumnType(cd.Type), PrimaryKey: cd.PrimaryKey}
	}
	return &TableSchemaInfo{Name: resp.Msg.Table, Columns: cols}, nil
}

// CreateTable creates a table with the given schema.
// The first column must be of type INT (it is the primary key).
func (c *Client) CreateTable(ctx context.Context, name string, schema Schema) error {
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", name, schema.String())
	result, err := c.Execute(ctx, sql)
	if err != nil {
		return err
	}
	_ = result
	return nil
}

// DropTable removes a table from the database.
func (c *Client) DropTable(ctx context.Context, name string) error {
	_, err := c.Execute(ctx, fmt.Sprintf("DROP TABLE %s", name))
	return err
}

// ── Fluent table accessor ─────────────────────────────────────────────────────

// Table returns a TableQuery builder for the named table.
func (c *Client) Table(name string) *TableQuery {
	return &TableQuery{client: c, tableName: name}
}

// ── Decode helpers ────────────────────────────────────────────────────────────

func decodeResultSet(rs *toydbv1.ResultSet) []Row {
	rows := make([]Row, len(rs.Rows))
	for i, pbRow := range rs.Rows {
		rows[i] = decodeRow(pbRow, rs.Columns)
	}
	return rows
}

func decodeRow(pbRow *toydbv1.Row, cols []string) Row {
	row := make(Row, len(cols))
	for _, col := range cols {
		if v, ok := pbRow.Values[col]; ok {
			row[col] = decodeValue(v)
		}
	}
	return row
}

func decodeValue(v *toydbv1.Value) any {
	if v == nil || v.IsNull {
		return nil
	}
	switch k := v.Kind.(type) {
	case *toydbv1.Value_IntVal:
		return k.IntVal
	case *toydbv1.Value_FloatVal:
		return k.FloatVal
	case *toydbv1.Value_TextVal:
		return k.TextVal
	case *toydbv1.Value_BoolVal:
		return k.BoolVal
	}
	return nil
}

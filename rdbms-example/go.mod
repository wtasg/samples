module rdbms

go 1.25.0

require (
	connectrpc.com/connect v1.20.0
	github.com/toydb/client v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.57.0
	google.golang.org/protobuf v1.36.11
)

require golang.org/x/text v0.40.0 // indirect

replace github.com/toydb/client => ../rdbms-client-lib-example

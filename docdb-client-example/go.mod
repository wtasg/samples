module docdb-client-app

go 1.25.0

require github.com/docdb/client v0.0.0

require (
	connectrpc.com/connect v1.20.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/docdb/client => ../docdb-client-lib-example

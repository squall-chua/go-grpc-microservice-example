# Go gRPC Microservice Example

A comprehensive, production-ready Golang gRPC microservice example demonstrating modern backend development practices. It features a fully-functional gRPC API alongside a REST API Gateway, MongoDB persistence utilizing [`gmqb`](https://github.com/squall-chua/gmqb) for dynamic queries, OAuth2 JWT token verification, and Prometheus metrics.

## Features

- **Protocol Buffers & gRPC**: Core API definitions and highly efficient RPC transport.
- **REST API Gateway**: HTTP/JSON to gRPC translation via `grpc-gateway`.
- **Multiplexing**: Runs both gRPC and HTTP servers on a single port (`8080`) securely.
- **MongoDB Persistence**: In-memory embedded MongoDB powered by `memongo` for zero-setup local execution.
- **Advanced Querying**: Dynamic MongoDB aggregation pipelines and filtering using `gmqb`, including precise `$facet` pagination support.
- **OAuth2 JWT Authentication**: Interceptor-based JSON Web Token verification parsing RS256/HS256 tokens using `golang-jwt`. 
- **Role-based Access Control (RBAC)**: Custom Protobuf annotations `(auth.v1.rule)` map required Scopes and Roles directly into the `proto` API schema.
- **Prometheus Metrics**: Automatic interceptor measuring of gRPC request rate, latency, and system performance via `/metrics`.
- **Graceful Shutdown**: Intercepts `SIGINT` and `SIGTERM` to cleanly stop standard HTTP requests and executing gRPC streams.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.24 or higher
- [Buf](https://buf.build/docs/installation) (for generating Protobuf stubs)
- `curl` (for testing endpoints)

## Creating the Protobuf Stubs

The project utilizes `buf` to automatically generate the Golang HTTP definitions, gRPC servers, and `ItemService` REST mappings from the `api/proto/v1/item.proto` definitions. It also utilizes `protoc-gen-openapiv2` to generate Swagger documentation from the defined schemas.

The required openapiv2 generator is tracked directly inside `go.mod` (using the Go 1.24+ `-tool` directive), ensuring reproducible builds without any manual binary installations.

To clean your environment and compile the `.pb.go` models and `.swagger.json` specs, run:

```bash
# Clean project dependencies
go mod tidy 

# Generate Protobuf stubs and OpenAPI spec
./scripts/generate.sh
```

This generates `item.pb.go`, `item.pb.gw.go`, `item_grpc.pb.go`, and `options.pb.go` inside the `api/v1` package, as well as `item.swagger.json` inside `api/swagger`.

## Running the Application

The server embeds an in-memory MongoDB instance via `memongo`, meaning you don't need a separate Mongo cluster to run or test the project!

Start the server:
```bash
go run ./cmd/server/main.go -port 8080 -cors-origins "*" -jwt-secret "super-secret-key"
```

To connect to an external MongoDB instance instead of the in-memory fallback, provide the URI:
```bash
go run ./cmd/server/main.go -mongo-uri "mongodb://admin:pass@localhost:27017/mydb"
```

You should see logs indicating MongoDB downloading/starting, followed by:
```
Starting Multiplexed gRPC & HTTP server on :8080
```

## Testing the Application

### 1. Generating a Test JWT

Because the application endpoints use Protobuf-level `<auth.v1.rule>` annotations, they require an `Authorization: Bearer <token>` header containing particular scopes/roles (e.g. `write:items` and `admin` status). 

The repository includes a helper utility to securely generate HMAC-signed test tokens. In a separate terminal, run:
```bash
export TOKEN=$(go run ./cmd/jwtgen/main.go -secret "super-secret-key" -scopes "read:items write:items" -roles "admin")
echo $TOKEN
```

### 2. Creating an Item (POST)
Uses `Write` privileges. Pass the outputted `$TOKEN` to verify Create permissions.
```bash
curl -X POST http://localhost:8080/v1/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Gaming Keyboard", "description": "Mechanical switches", "price": 129.99}'
```

### 3. List & Paginate Items (GET)
Uses `$facet` aggregation via `gmqb` to provide filtered pages alongside total document count.
```bash
# General List
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/items"

# Filter and Paginate
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/items?name_contains=Gaming&min_price=50&page_request.page_size=5&page_request.page_number=1"
```

### 4. Fetching Prometheus Metrics
A separate HTTP Mux provides Prometheus telemetry for performance monitoring.
```bash
curl http://localhost:8080/metrics
```

**Sample Output:**
```text
# HELP grpc_server_handled_total Total number of RPCs completed on the server, regardless of success or failure.
# TYPE grpc_server_handled_total counter
grpc_server_handled_total{grpc_code="OK",grpc_method="ListItems",grpc_service="item.v1.ItemService",grpc_type="unary"} 5
grpc_server_handled_total{grpc_code="OK",grpc_method="CreateItem",grpc_service="item.v1.ItemService",grpc_type="unary"} 2

# HELP grpc_server_started_total Total number of RPCs started on the server.
# TYPE grpc_server_started_total counter
grpc_server_started_total{grpc_method="ListItems",grpc_service="item.v1.ItemService",grpc_type="unary"} 5
grpc_server_started_total{grpc_method="CreateItem",grpc_service="item.v1.ItemService",grpc_type="unary"} 2

# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 18

# HELP items_created_total The total number of successfully created items
# TYPE items_created_total counter
items_created_total 2

# HELP items_deleted_total The total number of successfully deleted items
# TYPE items_deleted_total counter
items_deleted_total 1

# HELP items_updated_total The total number of successfully updated items
# TYPE items_updated_total counter
items_updated_total 0
```

### 5. Graceful Shutdown Validation
Press `Ctrl+C` in the terminal executing your server. You will observe graceful cleanup in action terminating all sub-processes:
```
Shutting down servers...
Servers gracefully stopped
```

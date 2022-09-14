# Amesh CP and DP gRPC API

## Dependency

Make sure the following prerequisites are satisfied if you want to modify the definitions in directory `proto`.

1. Install the [protoc-gen-validate](https://github.com/envoyproxy/protoc-gen-validate), and make sure the `protoc-gen-validate` binary is in your `$PATH`;
2. Install [googleapis](https://github.com/googleapis/googleapis);
3. Install [protobuf](https://github.com/protocolbuffers/protobuf), and make sure the `protoc` binary is in your `$PATH`;
4. Install [protoc-gen-go-grpc](https://grpc.io/docs/languages/go/quickstart/), and make sure the binary is in your `$PATH`;

## Code Generation (Go)

Run `make go` to generate codes in Go. Following environment variables should be set:

1. Set `PROTOC_GEN_VALIDATE_DIR` to the home directory of `protoc-gen-validate`;
2. Set `GOOGLE_PROTOBUF_DIR` to the home directory of `protobuf`;
3. Set `GOOGLEAPIS_DIR` to the home directory of `googleapis`;

```bash
make go PROTOC_GEN_VALIDATE_DIR=/path/to/protoc-gen-validate GOOGLE_PROTOBUF_DIR=/path/to/protobuf/src GOOGLEAPIS_DIR=/path/to/googleapis
```

After that, please enter the `go` directory and run `go mod tidy` to update the `go.mod` and `go.sum` files.

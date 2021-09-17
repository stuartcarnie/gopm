package rpc

//go:generate protoc -I. --go_out=. --go-grpc_out=. service.proto
//go:generate protoc -I. --js_out=:javascript --js_opt=import_style=commonjs service.proto
//go:generate protoc -I. --grpc-web_out=:javascript --grpc-web_opt=import_style=commonjs+dts,mode=grpcwebtext service.proto

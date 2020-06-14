package rpc

//go:generate protoc -I. --go_out=plugins=grpc:. service.proto
//go:generate protoc -I. --js_out=:../www --js_opt=import_style=commonjs service.proto
//go:generate protoc -I. --grpc-web_out=:../www --grpc-web_opt=import_style=commonjs+dts,mode=grpcwebtext service.proto

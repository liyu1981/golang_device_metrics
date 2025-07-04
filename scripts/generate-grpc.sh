#!/bin/bash

# Check if protoc is installed
if ! command -v protoc >/dev/null 2>&1; then
  echo "❌ Error: 'protoc' is not installed or not in your PATH."
  echo "👉 Please install it from https://grpc.io/docs/protoc-installation/"
  exit 1
fi

echo "✅ protoc found: $(protoc --version)"

# Install the Go plugin tools if not installed yet
echo "📦 Installing Go gRPC plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go files from proto
echo "🚀 Generating Go gRPC code..."
protoc --go_out=./pkg/grpc --go-grpc_out=./pkg/grpc pkg/grpc/service.proto

echo "✅ gRPC code generated in ./pkg/grpc"
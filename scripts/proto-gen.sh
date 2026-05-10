#!/bin/bash
# Generate Go code from protobuf definitions.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
#
# Install plugins:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$ROOT_DIR/proto/echofs/v1"
OUT_DIR="$ROOT_DIR/proto/gen/v1"

# Ensure output directory exists
mkdir -p "$OUT_DIR"

# Find protobuf include path
PROTO_INCLUDE=""
if command -v brew &> /dev/null; then
    PROTO_INCLUDE="$(brew --prefix protobuf 2>/dev/null)/include"
fi
if [ -z "$PROTO_INCLUDE" ] || [ ! -d "$PROTO_INCLUDE" ]; then
    PROTO_INCLUDE="/usr/local/include"
fi

echo "Generating proto code..."
echo "  Proto dir: $PROTO_DIR"
echo "  Output dir: $OUT_DIR"
echo "  Include: $PROTO_INCLUDE"

# Generate storage.proto
protoc \
    --proto_path="$PROTO_DIR" \
    --proto_path="$PROTO_INCLUDE" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR/storage.proto"

# Generate orchestrator.proto
protoc \
    --proto_path="$PROTO_DIR" \
    --proto_path="$PROTO_INCLUDE" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR/orchestrator.proto"

# Generate metadata.proto
protoc \
    --proto_path="$PROTO_DIR" \
    --proto_path="$PROTO_INCLUDE" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR/metadata.proto"

echo "Proto generation complete ✓"
echo "Generated files:"
ls -la "$OUT_DIR"/*.go

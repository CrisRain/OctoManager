// Package grpccodec provides a JSON-over-gRPC codec.
//
// The standard gRPC codec uses protobuf binary encoding. Registering this
// codec lets the Go worker communicate with Python plugin services using
// plain JSON payloads, which are human-readable and do not require protoc
// or grpcio-tools on the plugin side.
//
// The codec is registered globally by importing this package:
//
//	import _ "octomanger/internal/platform/grpccodec"
//
// Plugin gRPC clients must opt in per connection:
//
//	grpc.NewClient(addr, grpc.WithDefaultCallOptions(grpc.ForceCodec(grpccodec.Codec())))
package grpccodec

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/encoding"
)

const Name = "json"

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

// Codec returns the singleton JSON codec value (useful for ForceCodec calls).
func Codec() encoding.Codec {
	return jsonCodec{}
}

type jsonCodec struct{}

func (jsonCodec) Name() string { return Name }

// Marshal serialises a protobuf (or any json-tagged) message to JSON bytes.
// The generated protobuf Go types carry `json:"..."` struct tags, so this
// produces the standard proto3 JSON representation.
func (jsonCodec) Marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("grpccodec json marshal: %w", err)
	}
	return b, nil
}

// Unmarshal deserialises JSON bytes into a protobuf (or any json-tagged) message.
func (jsonCodec) Unmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("grpccodec json unmarshal: %w", err)
	}
	return nil
}

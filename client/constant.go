package client

const (
	mb = 1024 * 1024 // number of bytes in a megabyte

	// MaxPayloadSize is the maximum gRPC message size for all internal
	// service-to-service communication.
	MaxPayloadSize = 2048 * mb
)

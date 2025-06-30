package client

const (
	mb = 1024 * 1024 // number of bytes in a megabyte

	// MaxPayloadSize is the maximum size of the payload that grpc clients allow.
	MaxPayloadSize = 256 * mb
)

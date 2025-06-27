package client

// HTTPSConfig contains the details to establish a secure HTTP connection.
type HTTPSConfig struct {
	Cert string `koanf:"cert"`
	Key  string `koanf:"key"`
}

// ServiceConfig contains the details needed to connect with a service.
type ServiceConfig struct {
	Host        string      `koanf:"host"`
	PublicPort  int         `koanf:"publicport"`
	PrivatePort int         `koanf:"privateport"`
	HTTPS       HTTPSConfig `koanf:"https"`
}

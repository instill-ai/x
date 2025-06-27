package client

// ServiceConfig contains the details needed to connect with a service.
type ServiceConfig struct {
	Host        string `koanf:"host"`
	PublicPort  int    `koanf:"publicport"`
	PrivatePort int    `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

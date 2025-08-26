package openfga

// Config contains the configuration parameters for OpenFGA.
// Simplified without replica configuration.
type Config struct {
	Host string `koanf:"host"`
	Port int    `koanf:"port"`
}

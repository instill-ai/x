package minio

// Config is the minio configuration.
type Config struct {
	Host       string `koanf:"host"`
	Port       string `koanf:"port"`
	User       string `koanf:"user"`
	Password   string `koanf:"password"`
	BucketName string `koanf:"bucketname"`
	Secure     bool   `koanf:"secure"` // Add this line for the Secure option
}

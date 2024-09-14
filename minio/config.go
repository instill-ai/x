package minio

// Config is the minio configuration.
type Config struct {
	Host       string `koanf:"host"`
	Port       string `koanf:"port"`
	RootUser   string `koanf:"rootuser"`
	RootPwd    string `koanf:"rootpwd"`
	BucketName string `koanf:"bucketname"`
	Secure     bool   `koanf:"secure"` // Add this line for the Secure option
}

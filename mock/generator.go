package mock

// client package
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.ClientCreator -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.ConnectionManager -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.ClientFactory -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.TLSProvider -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.MetadataPropagator -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.Option -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.Options -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.ServiceConfig -o ./client -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/client.HTTPSConfig -o ./client -s "_mock.gen.go"

// minio package
//go:generate minimock -g -i github.com/instill-ai/x/minio.Client -o ./minio -s "_mock.gen.go"

// server package
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.Logger -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.OTELLogger -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.ServerStream -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.Marshaler -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.Decoder -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.Encoder -o ./server -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/server.ProtoMessage -o ./server -s "_mock.gen.go"

// log package
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.Logger -o ./log -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.OTELLogger -o ./log -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.Encoder -o ./log -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.LoggerFactory -o ./log -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.CoreFactory -o ./log -s "_mock.gen.go"
//go:generate minimock -g -i github.com/instill-ai/x/mock/log.SyncerFactory -o ./log -s "_mock.gen.go"

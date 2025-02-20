package mock

//go:generate minimock -g -i github.com/instill-ai/x/minio.Client -o ./ -s "_mock.gen.go"

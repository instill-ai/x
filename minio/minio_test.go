package minio_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	miniox "github.com/instill-ai/x/minio"
)

func TestMinio(t *testing.T) {
	t.Skipf("only for testing on local")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, _ := zap.NewDevelopment()
	mc, err := miniox.NewMinioClientAndInitBucket(ctx, &miniox.Config{
		Host:       "localhost",
		Port:       "19000",
		RootUser:   "minioadmin",
		RootPwd:    "minioadmin",
		BucketName: "instill-ai-model",
	}, log)

	require.NoError(t, err)

	t.Log("test upload file to minio")
	fileName, _ := uuid.NewV4()
	uid, _ := uuid.NewV4()

	data := make(map[string]string)
	data["uid"] = uid.String()
	jsonBytes, _ := json.Marshal(data)

	url, stat, err := mc.UploadFile(ctx, log, fileName.String(), data, "application/json")
	require.NoError(t, err)
	t.Log("url:", url)
	t.Log("size:", stat.Size)

	fileBytes, err := mc.GetFile(ctx, log, fileName.String())
	require.NoError(t, err)
	require.Equal(t, jsonBytes, fileBytes)
}

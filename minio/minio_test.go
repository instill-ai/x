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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, _ := zap.NewDevelopment()
	params := miniox.ClientParams{
		Logger: log,
		Config: miniox.Config{
			Host:       "localhost",
			Port:       "19000",
			User:       "minioadmin",
			Password:   "minioadmin",
			BucketName: "instill-ai-model",
		},
	}
	mc, err := miniox.NewMinioClientAndInitBucket(ctx, params)

	require.NoError(t, err)

	t.Log("test upload file to minio")
	fileName, _ := uuid.NewV4()

	userUID := uuid.Must(uuid.NewV4())
	data := make(map[string]string)
	data["uid"] = uuid.Must(uuid.NewV4()).String()
	jsonBytes, _ := json.Marshal(data)

	url, stat, err := mc.UploadFile(ctx, &miniox.UploadFileParam{
		UserUID:      userUID,
		FilePath:     fileName.String(),
		FileContent:  data,
		FileMimeType: "application/json",
	})
	require.NoError(t, err)
	t.Log("url:", url)
	t.Log("size:", stat.Size)

	fileBytes, err := mc.GetFile(ctx, userUID, fileName.String())
	require.NoError(t, err)
	require.Equal(t, jsonBytes, fileBytes)

	err = mc.DeleteFile(ctx, userUID, fileName.String())
	require.NoError(t, err)

	_, err = mc.GetFile(ctx, userUID, fileName.String())
	require.Error(t, err)
}

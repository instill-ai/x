package blobstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func UploadFile(ctx context.Context, logger *zap.Logger, uploadURL string, data []byte, contentType string) error {

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, nil)

	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	body := bytes.NewReader(data)
	contentLength := int64(len(data))
	req.Body = io.NopCloser(body)
	req.Header = metadataToHTTPHeaders(ctx)

	req.ContentLength = contentLength
	req.Header.Set("Content-Type", contentType)
	req.Header.Del("Authorization")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("uploading blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to upload file")
		logger.Error("Failed to upload file to MinIO",
			zap.Binary("body", body),
			zap.Int("status", resp.StatusCode),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func metadataToHTTPHeaders(ctx context.Context) http.Header {
	headers := http.Header{}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return headers
	}
	for key, values := range md {
		for _, value := range values {
			headers.Add(key, value)
		}
	}
	return headers
}

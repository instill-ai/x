package minio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"go.uber.org/zap"

	miniogo "github.com/minio/minio-go/v7"
)

type MinioI interface {
	UploadFile(ctx context.Context, logger *zap.Logger, param *UploadFileParam) (url string, objectInfo *miniogo.ObjectInfo, err error)
	UploadFileBytes(ctx context.Context, logger *zap.Logger, param *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error)
	DeleteFile(ctx context.Context, logger *zap.Logger, filePath string) (err error)
	GetFile(ctx context.Context, logger *zap.Logger, filePath string) ([]byte, error)
	GetFilesByPaths(ctx context.Context, logger *zap.Logger, filePaths []string) ([]FileContent, error)
}

type ExpiryGroup string

func (g ExpiryGroup) String() string {
	return string(g)
}

const (
	Location = "us-east-1"

	StatusEnabled = "Enabled"
	ExpiryTag     = "expiry-group"

	FreePlanExpiry       ExpiryGroup = "free-plan-expiry"
	ProPlanExpiry        ExpiryGroup = "pro-plan-expiry"
	TeamPlanExpiry       ExpiryGroup = "team-plan-expiry"
	EnterprisePlanExpiry ExpiryGroup = "enterprise-plan-expiry"
)

var expiryTimeConfig = map[ExpiryGroup]int{
	FreePlanExpiry: 3,
	ProPlanExpiry:  7,
	TeamPlanExpiry: 30,
}

type minio struct {
	client *miniogo.Client
	bucket string
}

func NewMinioClientAndInitBucket(ctx context.Context, cfg *Config, logger *zap.Logger) (MinioI, error) {
	logger.Info("Initializing Minio client and bucket...")

	endpoint := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.RootUser, cfg.RootPwd, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		logger.Error("cannot connect to minio",
			zap.String("host:port", cfg.Host+":"+cfg.Port),
			zap.String("user", cfg.RootUser),
			zap.String("pwd", cfg.RootPwd), zap.Error(err))
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		logger.Error("failed in checking BucketExists", zap.Error(err))
		return nil, err
	}
	if exists {
		logger.Info("Bucket already exists", zap.String("bucket", cfg.BucketName))
		return &minio{client: client, bucket: cfg.BucketName}, nil
	}

	if err = client.MakeBucket(ctx, cfg.BucketName, miniogo.MakeBucketOptions{
		Region: Location,
	}); err != nil {
		logger.Error("creating Bucket failed", zap.Error(err))
		return nil, err
	}
	logger.Info("Successfully created bucket", zap.String("bucket", cfg.BucketName))

	lccfg := lifecycle.NewConfiguration()
	lccfg.Rules = []lifecycle.Rule{
		{
			ID:     "expire-bucket-objects",
			Status: StatusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(30),
			},
		},
		{
			ID:     "free-plan-rule",
			Status: StatusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(expiryTimeConfig[FreePlanExpiry]),
			},
			RuleFilter: lifecycle.Filter{
				Tag: lifecycle.Tag{
					Key:   ExpiryTag,
					Value: FreePlanExpiry.String(),
				},
			},
		},
		{
			ID:     "pro-plan-rule",
			Status: StatusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(expiryTimeConfig[ProPlanExpiry]),
			},
			RuleFilter: lifecycle.Filter{
				Tag: lifecycle.Tag{
					Key:   ExpiryTag,
					Value: ProPlanExpiry.String(),
				},
			},
		},
		{
			ID:     "team-plan-rule",
			Status: StatusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(expiryTimeConfig[TeamPlanExpiry]),
			},
			RuleFilter: lifecycle.Filter{
				Tag: lifecycle.Tag{
					Key:   ExpiryTag,
					Value: TeamPlanExpiry.String(),
				},
			},
		},
	}

	err = client.SetBucketLifecycle(ctx, cfg.BucketName, lccfg)
	if err != nil {
		logger.Error("setting Bucket lifecycle failed", zap.Error(err))
		return nil, err
	}

	return &minio{client: client, bucket: cfg.BucketName}, nil
}

type UploadFileParam struct {
	FilePath     string
	FileContent  any
	FileMimeType string
	ExpiryGroup  ExpiryGroup
}

type UploadFileBytesParam struct {
	FilePath     string
	FileBytes    []byte
	FileMimeType string
	ExpiryGroup  ExpiryGroup
}

func (m *minio) UploadFile(ctx context.Context, logger *zap.Logger, param *UploadFileParam) (url string, objectInfo *miniogo.ObjectInfo, err error) {
	jsonData, _ := json.Marshal(param.FileContent)
	return m.UploadFileBytes(ctx, logger, &UploadFileBytesParam{
		FilePath:     param.FilePath,
		FileBytes:    jsonData,
		FileMimeType: param.FileMimeType,
		ExpiryGroup:  param.ExpiryGroup,
	})
}

func (m *minio) UploadFileBytes(ctx context.Context, logger *zap.Logger, param *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error) {
	logger.Info("start to upload file to minio", zap.String("filePath", param.FilePath))
	reader := bytes.NewReader(param.FileBytes)

	// Create the file path with folder structure
	_, err = m.client.PutObject(ctx, m.bucket, param.FilePath, reader, int64(len(param.FileBytes)), miniogo.PutObjectOptions{
		ContentType: param.FileMimeType,
		UserTags:    map[string]string{ExpiryTag: param.ExpiryGroup.String()},
	})
	if err != nil {
		logger.Error("Failed to upload file to MinIO", zap.Error(err))
		return "", nil, err
	}

	// Get the object stat (metadata)
	stat, err := m.client.StatObject(ctx, m.bucket, param.FilePath, miniogo.StatObjectOptions{})
	if err != nil {
		return "", nil, err
	}

	// Generate the presigned URL
	expiryDays, ok := expiryTimeConfig[param.ExpiryGroup]
	if !ok {
		expiryDays = 30
	}
	expiryDuration := time.Hour * 24 * time.Duration(expiryDays)
	presignedURL, err := m.client.PresignedGetObject(ctx, m.bucket, param.FilePath, expiryDuration, nil)
	if err != nil {
		return "", nil, err
	}

	return presignedURL.String(), &stat, nil
}

// DeleteFile delete the file from minio
func (m *minio) DeleteFile(ctx context.Context, logger *zap.Logger, filePath string) (err error) {
	err = m.client.RemoveObject(ctx, m.bucket, filePath, miniogo.RemoveObjectOptions{})
	if err != nil {
		logger.Error("Failed to delete file from MinIO", zap.Error(err))
		return err
	}
	return nil
}

// GetFile Get the object using the client
func (m *minio) GetFile(ctx context.Context, logger *zap.Logger, filePath string) ([]byte, error) {
	object, err := m.client.GetObject(ctx, m.bucket, filePath, miniogo.GetObjectOptions{})
	if err != nil {
		logger.Error("Failed to get file from MinIO", zap.Error(err))
		return nil, err
	}
	defer object.Close()

	// Read the object's content
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		logger.Error("Failed to read file from MinIO", zap.Error(err))
		return nil, err
	}

	return buf.Bytes(), nil
}

// FileContent represents a file and its content
type FileContent struct {
	Name    string
	Content []byte
}

// GetFilesByPaths GetFiles retrieves the contents of specified files from MinIO
func (m *minio) GetFilesByPaths(ctx context.Context, logger *zap.Logger, filePaths []string) ([]FileContent, error) {
	var wg sync.WaitGroup
	fileCount := len(filePaths)

	errCh := make(chan error, fileCount)
	resultCh := make(chan FileContent, fileCount)

	for _, path := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			obj, err := m.client.GetObject(ctx, m.bucket, filePath, miniogo.GetObjectOptions{})
			if err != nil {
				logger.Error("Failed to get object from MinIO", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}
			defer obj.Close()

			var buffer bytes.Buffer
			_, err = io.Copy(&buffer, obj)
			if err != nil {
				logger.Error("Failed to read object content", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}

			fileContent := FileContent{
				Name:    filePath,
				Content: buffer.Bytes(),
			}
			resultCh <- fileContent
		}(path)
	}

	wg.Wait()

	close(errCh)
	close(resultCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	files := make([]FileContent, 0)
	for fileContent := range resultCh {
		files = append(files, fileContent)
	}

	return files, nil
}

func GenerateInputRefID(prefix string) string {
	referenceUID, _ := uuid.NewV4()
	return prefix + "/input/" + referenceUID.String()
}

func GenerateOutputRefID(prefix string) string {
	referenceUID, _ := uuid.NewV4()
	return prefix + "/output/" + referenceUID.String()
}

package minio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"go.uber.org/zap"

	miniogo "github.com/minio/minio-go/v7"
)

// MinIOHeaderUserUID is sent as metadata in MinIO request headers to indicate
// the user that triggered the action.
const MinIOHeaderUserUID = "x-amz-meta-instill-user-uid"

// MinioI defines the methods to interact with MinIO.
type MinioI interface {
	WithLogger(*zap.Logger) MinioI
	UploadFile(context.Context, *UploadFileParam) (url string, objectInfo *miniogo.ObjectInfo, err error)
	UploadFileBytes(context.Context, *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error)
	DeleteFile(ctx context.Context, userUID uuid.UUID, filePath string) (err error)
	GetFile(ctx context.Context, userUID uuid.UUID, filePath string) ([]byte, error)
	GetFilesByPaths(ctx context.Context, userUID uuid.UUID, filePaths []string) ([]FileContent, error)
}

type ExpiryRule struct {
	Tag            string
	ExpirationDays int
}

const (
	Location = "us-east-1"

	StatusEnabled = "Enabled"
	ExpiryTag     = "expiry-group"
)

type minio struct {
	client           *miniogo.Client
	bucket           string
	expiryRuleConfig map[string]int

	// logger will be used to audit who tries to perform an action on the MinIO
	// data.
	logger *zap.Logger
}

// AppInfo contains the name and version of the requester.
type AppInfo struct {
	Name    string
	Version string
}

// ClientParams contains the information required to initialize a MinIO
// client.
type ClientParams struct {
	Config      Config
	Logger      *zap.Logger
	ExpiryRules []ExpiryRule
	AppInfo     AppInfo
}

// NewMinioClientAndInitBucket initializes a MinIO bucket (creating it if it
// doesn't exist and applying the lifecycle rules specified in the
// configuration) and returns a client to interact with such bucket.
func NewMinioClientAndInitBucket(ctx context.Context, params ClientParams) (MinioI, error) {
	cfg := params.Config
	logger := params.Logger.With(zap.String("bucket", cfg.BucketName))
	logger.Info("Initializing MinIO client and bucket")

	endpoint := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.RootUser, cfg.RootPwd, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to MinIO: %w", err)
	}

	client.SetAppInfo(params.AppInfo.Name, params.AppInfo.Version)

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("checking bucket existence: %w", err)
	}

	if !exists {
		if err = client.MakeBucket(ctx, cfg.BucketName, miniogo.MakeBucketOptions{
			Region: Location,
		}); err != nil {
			return nil, fmt.Errorf("creating bucket: %w", err)
		}
		logger.Info("Successfully created bucket")
	} else {
		logger.Info("Bucket already exists")
	}

	lccfg := lifecycle.NewConfiguration()
	lccfg.Rules = make([]lifecycle.Rule, 0, len(params.ExpiryRules))

	expiryRuleConfig := make(map[string]int)
	for _, expiryRule := range params.ExpiryRules {
		expiryRuleConfig[expiryRule.Tag] = expiryRule.ExpirationDays
		if expiryRule.ExpirationDays <= 0 {
			// On MinIO, we can define expiration rules for tags, but we can't
			// set a "no expiration" rule. Clients, however, might want to have
			// such rules for certain objects. A 0 expiration day rule means no
			// expiration. We won't set this rule but we'll keep the tag in the
			// `expiryRuleConfig` object.
			logger.Info(
				"Skipping lifecycle rule - tag will have infinite retention",
				zap.String("tag", expiryRule.Tag),
			)
			continue
		}

		lccfg.Rules = append(lccfg.Rules, lifecycle.Rule{
			ID:     expiryRule.Tag,
			Status: StatusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(expiryRule.ExpirationDays),
			},
			RuleFilter: lifecycle.Filter{
				Tag: lifecycle.Tag{
					Key:   ExpiryTag,
					Value: expiryRule.Tag,
				},
			},
		})
	}

	err = client.SetBucketLifecycle(ctx, cfg.BucketName, lccfg)
	if err != nil {
		return nil, fmt.Errorf("applying lifecycle rules: %w", err)
	}

	return &minio{
		client:           client,
		bucket:           cfg.BucketName,
		expiryRuleConfig: expiryRuleConfig,
		logger:           logger,
	}, nil
}

// WithLogger returns a copy of the MinIO client with the provided logger.
func (m *minio) WithLogger(log *zap.Logger) MinioI {
	return &minio{
		client:           m.client,
		bucket:           m.bucket,
		expiryRuleConfig: m.expiryRuleConfig,
		logger:           log.With(zap.String("bucket", m.bucket)),
	}
}

// UploadFileParam contains the information to upload a file to MinIO.
type UploadFileParam struct {
	UserUID       uuid.UUID
	FilePath      string
	FileContent   any
	FileMimeType  string
	ExpiryRuleTag string
}

// UploadFileBytesParam contains the information to upload a data blob to
// MinIO.
type UploadFileBytesParam struct {
	UserUID       uuid.UUID
	FilePath      string
	FileBytes     []byte
	FileMimeType  string
	ExpiryRuleTag string
}

func (m *minio) UploadFile(ctx context.Context, param *UploadFileParam) (url string, objectInfo *miniogo.ObjectInfo, err error) {
	jsonData, _ := json.Marshal(param.FileContent)
	return m.UploadFileBytes(ctx, &UploadFileBytesParam{
		UserUID:       param.UserUID,
		FilePath:      param.FilePath,
		FileBytes:     jsonData,
		FileMimeType:  param.FileMimeType,
		ExpiryRuleTag: param.ExpiryRuleTag,
	})
}

func (m *minio) UploadFileBytes(ctx context.Context, param *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error) {
	reader := bytes.NewReader(param.FileBytes)

	// Create the file path with folder structure
	_, err = m.client.PutObject(ctx,
		m.bucket,
		param.FilePath,
		reader,
		int64(len(param.FileBytes)),
		miniogo.PutObjectOptions{
			ContentType:  param.FileMimeType,
			UserTags:     map[string]string{ExpiryTag: param.ExpiryRuleTag},
			UserMetadata: map[string]string{MinIOHeaderUserUID: param.UserUID.String()},
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("putting object in MinIO: %w", err)
	}

	// Get the object stat (metadata)
	statOpts := miniogo.StatObjectOptions(m.getObjectOptions(param.UserUID))
	stat, err := m.client.StatObject(ctx, m.bucket, param.FilePath, statOpts)
	if err != nil {
		return "", nil, fmt.Errorf("getting object stats: %w", err)
	}

	// Generate the presigned URL
	expiryDays, ok := m.expiryRuleConfig[param.ExpiryRuleTag]
	if !ok || expiryDays <= 0 || expiryDays > 7 { // presignedURL Expires cannot be greater than 7 days.
		expiryDays = 7
	}
	expiryDuration := time.Hour * 24 * time.Duration(expiryDays)

	// We're using PresignHeader in order to be able to pass the user UID in
	// the request. If PresignedGetObject supports this at any point, we should
	// use that method.
	presignedURL, err := m.client.PresignHeader(
		ctx,
		http.MethodGet,
		m.bucket,
		param.FilePath,
		expiryDuration,
		nil,
		statOpts.Header(),
	)
	if err != nil {
		return "", nil, fmt.Errorf("getting presigned object URL: %w", err)
	}

	return presignedURL.String(), &stat, nil
}

// DeleteFile delete the file frotom minio
func (m *minio) DeleteFile(ctx context.Context, userUID uuid.UUID, filePath string) (err error) {
	// MinIO (and S3) API doesn't expose a way to pass headers to the deletion
	// method. The client will be responsible of logging this information.
	log := m.logger.With(
		zap.String("path", filePath),
		zap.String("userUID", userUID.String()),
	)
	log.Info("Object deletion in MinIO")

	err = m.client.RemoveObject(ctx, m.bucket, filePath, miniogo.RemoveObjectOptions{})
	if err != nil {
		log.Error("Failed to delete file from MinIO", zap.Error(err))
		return fmt.Errorf("removing object in MinIO: %w", err)
	}

	return nil
}

// GetFile Get the object using the client
func (m *minio) GetFile(ctx context.Context, userUID uuid.UUID, filePath string) ([]byte, error) {
	object, err := m.client.GetObject(ctx, m.bucket, filePath, m.getObjectOptions(userUID))
	if err != nil {
		return nil, fmt.Errorf("getting object from MinIO: %w", err)
	}
	defer object.Close()

	// Read the object's content
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		return nil, fmt.Errorf("reading MinIO object: %w", err)
	}

	return buf.Bytes(), nil
}

// FileContent represents a file and its content
type FileContent struct {
	Name    string
	Content []byte
}

// GetFilesByPaths GetFiles retrieves the contents of specified files from MinIO
func (m *minio) GetFilesByPaths(ctx context.Context, userUID uuid.UUID, filePaths []string) ([]FileContent, error) {
	var wg sync.WaitGroup
	fileCount := len(filePaths)

	errCh := make(chan error, fileCount)
	resultCh := make(chan FileContent, fileCount)

	for _, path := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			obj, err := m.GetFile(ctx, userUID, filePath)
			if err != nil {
				errCh <- err
				return
			}

			fileContent := FileContent{
				Name:    filePath,
				Content: obj,
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

// It's used for the operations that don't need to initialize a bucket
// We will refactor the MinIO shared logic in the future
type minioClientWrapper struct {
	client *miniogo.Client
}

// NewMinioClient returns a new MinIO client.
func NewMinioClient(ctx context.Context, cfg *Config, logger *zap.Logger) (*minioClientWrapper, error) {
	logger.Info("Initializing MinIO client")

	endpoint := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.RootUser, cfg.RootPwd, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to MinIO: %w", err)
	}

	return &minioClientWrapper{client: client}, nil
}

// GetFile fetches a file from minio
func (m *minioClientWrapper) GetFile(ctx context.Context, bucketName, objectPath string) (data []byte, contentType string, err error) {
	object, err := m.client.GetObject(ctx, bucketName, objectPath, miniogo.GetObjectOptions{})

	if err != nil {
		return nil, "", fmt.Errorf("get object: %w", err)
	}

	defer object.Close()

	info, err := object.Stat()

	if err != nil {
		return nil, "", fmt.Errorf("get object info: %w", err)
	}
	data, err = io.ReadAll(object)

	if err != nil {
		return nil, "", fmt.Errorf("read object: %w", err)
	}

	return data, info.ContentType, nil
}

func (m *minio) getObjectOptions(userUID uuid.UUID) miniogo.GetObjectOptions {
	opts := miniogo.GetObjectOptions{}
	opts.Set(MinIOHeaderUserUID, userUID.String())
	return opts
}

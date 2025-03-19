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

// Client defines the methods to interact with MinIO.
type Client interface {
	// WithLogger sets the Client logger.
	WithLogger(*zap.Logger) Client

	// UploadPrivateFileBytes uploads a data blob to be used internally. The
	// uploaded object will only be accessible through its file path and its
	// URL won't be publicly exposed.
	UploadPrivateFileBytes(context.Context, UploadFileBytesParam) error

	// UploadFile[Bytes] uploads a data blob and returns the object information
	// and a presigned URL to access it publicly.
	UploadFile(context.Context, *UploadFileParam) (url string, objectInfo *miniogo.ObjectInfo, err error)
	UploadFileBytes(context.Context, *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error)

	DeleteFile(ctx context.Context, userUID uuid.UUID, filePath string) (err error)
	GetFile(ctx context.Context, userUID uuid.UUID, filePath string) ([]byte, error)
	GetFilesByPaths(ctx context.Context, userUID uuid.UUID, filePaths []string) ([]FileContent, error)

	// Client returns MinIO's SDK client. This is used to migrate progressively
	// services like `artifact-backend` to adopt this package. Eventually, the
	// SDK client shouldn't be exposed and services should only use public
	// methods in this package.
	Client() *miniogo.Client
}

// ExpiryRule defines an expiration policy for tagged objects.
type ExpiryRule struct {
	Tag            string
	ExpirationDays int
}

const (
	// Location is the default location of MinIO buckets.
	Location = "us-east-1"
	// MinIOHeaderUserUID is sent as metadata in MinIO request headers to indicate
	// the user that triggered the action.
	MinIOHeaderUserUID = "x-amz-meta-instill-user-uid"

	statusEnabled = "Enabled"
	expiryTag     = "expiry-group"
)

// GenerateInputRefID returns a prefixed object path or an input file.
func GenerateInputRefID(prefix string) string {
	referenceUID, _ := uuid.NewV4()
	return prefix + "/input/" + referenceUID.String()
}

// GenerateOutputRefID returns a prefixed object path or an ouptut file.
func GenerateOutputRefID(prefix string) string {
	referenceUID, _ := uuid.NewV4()
	return prefix + "/output/" + referenceUID.String()
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

// FileGetter fetches files from the MinIO blob storage.
type FileGetter struct {
	client *miniogo.Client
	logger *zap.Logger
}

// NewFileGetter returns a MinIO client that is able to fetch files.
func NewFileGetter(params ClientParams) (*FileGetter, error) {
	logger := params.Logger
	logger.Info("Initializing MinIO client")

	client, err := newClient(params)
	if err != nil {
		return nil, err
	}

	return &FileGetter{
		client: client,
		logger: logger,
	}, nil
}

// GetFileParams contains the information to fetch a file from MinIO.
type GetFileParams struct {
	// UserUID is the authenticated user that's requesting the file.
	UserUID    uuid.UUID
	BucketName string
	Path       string
}

// GetFile fetches a file from MinIO.
func (fg *FileGetter) GetFile(ctx context.Context, p GetFileParams) (data []byte, contentType string, err error) {
	object, err := fg.client.GetObject(ctx, p.BucketName, p.Path, getObjectOptions(p.UserUID))
	if err != nil {
		return nil, "", fmt.Errorf("fetching object: %w", err)
	}
	defer object.Close()

	info, err := object.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("getting object info: %w", err)
	}

	data, err = io.ReadAll(object)
	if err != nil {
		return nil, "", fmt.Errorf("reading object: %w", err)
	}

	return data, info.ContentType, nil
}

type minio struct {
	client           *miniogo.Client
	bucket           string
	expiryRuleConfig map[string]int
	logger           *zap.Logger
}

// NewMinIOClientAndInitBucket initializes a MinIO bucket (creating it if it
// doesn't exist) applies the lifecycle rules specified in the configuration
// and returns a client to interact with such bucket.
func NewMinIOClientAndInitBucket(ctx context.Context, params ClientParams) (Client, error) {
	cfg := params.Config
	logger := params.Logger.With(zap.String("bucket", cfg.BucketName))
	logger.Info("Initializing MinIO client and bucket")

	client, err := newClient(params)
	if err != nil {
		return nil, err
	}

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
			Status: statusEnabled,
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(expiryRule.ExpirationDays),
			},
			RuleFilter: lifecycle.Filter{
				Tag: lifecycle.Tag{
					Key:   expiryTag,
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
func (m *minio) WithLogger(log *zap.Logger) Client {
	return &minio{
		client:           m.client,
		bucket:           m.bucket,
		expiryRuleConfig: m.expiryRuleConfig,
		logger:           log.With(zap.String("bucket", m.bucket)),
	}
}

// Client returns the internal MinIO SDK client.
func (m *minio) Client() *miniogo.Client { return m.client }

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

func (m *minio) UploadPrivateFileBytes(ctx context.Context, param UploadFileBytesParam) error {
	_, err := m.client.PutObject(ctx,
		m.bucket,
		param.FilePath,
		bytes.NewReader(param.FileBytes),
		int64(len(param.FileBytes)),
		miniogo.PutObjectOptions{
			ContentType:  param.FileMimeType,
			UserTags:     map[string]string{expiryTag: param.ExpiryRuleTag},
			UserMetadata: map[string]string{MinIOHeaderUserUID: param.UserUID.String()},
		},
	)

	return err
}

func (m *minio) UploadFileBytes(ctx context.Context, param *UploadFileBytesParam) (url string, objectInfo *miniogo.ObjectInfo, err error) {
	if err := m.UploadPrivateFileBytes(ctx, *param); err != nil {
		return "", nil, fmt.Errorf("putting object in MinIO: %w", err)
	}

	// Get the object stat (metadata)
	statOpts := miniogo.StatObjectOptions(getObjectOptions(param.UserUID))
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
	object, err := m.client.GetObject(ctx, m.bucket, filePath, getObjectOptions(userUID))
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

func newClient(params ClientParams) (*miniogo.Client, error) {
	cfg := params.Config

	endpoint := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.User, cfg.Password, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to MinIO: %w", err)
	}

	client.SetAppInfo(params.AppInfo.Name, params.AppInfo.Version)

	return client, nil
}

func getObjectOptions(userUID uuid.UUID) miniogo.GetObjectOptions {
	opts := miniogo.GetObjectOptions{}
	opts.Set(MinIOHeaderUserUID, userUID.String())
	return opts
}

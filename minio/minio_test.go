package minio_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"go.uber.org/zap"

	miniogo "github.com/minio/minio-go/v7"

	miniox "github.com/instill-ai/x/minio"
	mockminio "github.com/instill-ai/x/mock/minio"
)

func TestMinioClient_WithLogger(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	// Set up mock expectation
	newLogger := zap.NewNop()
	mockClient.WithLoggerMock.Expect(newLogger).Return(mockClient)

	// Test the method
	result := mockClient.WithLogger(newLogger)

	// Verify the result
	qt.Check(result, quicktest.Not(quicktest.IsNil))
}

func TestMinioClient_UploadFile(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"
	fileContent := map[string]string{"test": "data"}
	fileMimeType := "application/json"

	// Marshal the content to JSON
	jsonBytes, _ := json.Marshal(fileContent)

	// Create upload parameter
	uploadParam := &miniox.UploadFileParam{
		UserUID:      userUID,
		FilePath:     filePath,
		FileContent:  fileContent,
		FileMimeType: fileMimeType,
	}

	// Expected results
	expectedURL := "https://example.com/presigned-url"
	expectedObjectInfo := &miniogo.ObjectInfo{
		Size: int64(len(jsonBytes)),
	}

	// Set up mock expectations
	mockClient.UploadFileMock.Expect(ctx, uploadParam).Return(expectedURL, expectedObjectInfo, nil)

	// Test the method
	url, objectInfo, err := mockClient.UploadFile(ctx, uploadParam)

	// Verify results
	qt.Check(err, quicktest.IsNil)
	qt.Check(url, quicktest.Equals, expectedURL)
	qt.Check(objectInfo, quicktest.Equals, expectedObjectInfo)
}

func TestMinioClient_UploadFile_Error(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"
	fileContent := map[string]string{"test": "data"}
	fileMimeType := "application/json"

	// Create upload parameter
	uploadParam := &miniox.UploadFileParam{
		UserUID:      userUID,
		FilePath:     filePath,
		FileContent:  fileContent,
		FileMimeType: fileMimeType,
	}

	// Expected error
	expectedError := errors.New("upload failed")

	// Set up mock expectations
	mockClient.UploadFileMock.Expect(ctx, uploadParam).Return("", nil, expectedError)

	// Test the method
	url, objectInfo, err := mockClient.UploadFile(ctx, uploadParam)

	// Verify results
	qt.Check(err, quicktest.Equals, expectedError)
	qt.Check(url, quicktest.Equals, "")
	qt.Check(objectInfo, quicktest.IsNil)
}

func TestMinioClient_UploadFileBytes(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"
	fileBytes := []byte(`{"test": "data"}`)
	fileMimeType := "application/json"

	// Create upload parameter
	uploadParam := &miniox.UploadFileBytesParam{
		UserUID:      userUID,
		FilePath:     filePath,
		FileBytes:    fileBytes,
		FileMimeType: fileMimeType,
	}

	// Expected results
	expectedURL := "https://example.com/presigned-url"
	expectedObjectInfo := &miniogo.ObjectInfo{
		Size: int64(len(fileBytes)),
	}

	// Set up mock expectations
	mockClient.UploadFileBytesMock.Expect(ctx, uploadParam).Return(expectedURL, expectedObjectInfo, nil)

	// Test the method
	url, objectInfo, err := mockClient.UploadFileBytes(ctx, uploadParam)

	// Verify results
	qt.Check(err, quicktest.IsNil)
	qt.Check(url, quicktest.Equals, expectedURL)
	qt.Check(objectInfo, quicktest.Equals, expectedObjectInfo)
}

func TestMinioClient_UploadPrivateFileBytes(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/private/file.json"
	fileBytes := []byte(`{"private": "data"}`)
	fileMimeType := "application/json"

	// Create upload parameter
	uploadParam := miniox.UploadFileBytesParam{
		UserUID:      userUID,
		FilePath:     filePath,
		FileBytes:    fileBytes,
		FileMimeType: fileMimeType,
	}

	// Set up mock expectations
	mockClient.UploadPrivateFileBytesMock.Expect(ctx, uploadParam).Return(nil)

	// Test the method
	err := mockClient.UploadPrivateFileBytes(ctx, uploadParam)

	// Verify results
	qt.Check(err, quicktest.IsNil)
}

func TestMinioClient_UploadPrivateFileBytes_Error(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/private/file.json"
	fileBytes := []byte(`{"private": "data"}`)
	fileMimeType := "application/json"

	// Create upload parameter
	uploadParam := miniox.UploadFileBytesParam{
		UserUID:      userUID,
		FilePath:     filePath,
		FileBytes:    fileBytes,
		FileMimeType: fileMimeType,
	}

	// Expected error
	expectedError := errors.New("private upload failed")

	// Set up mock expectations
	mockClient.UploadPrivateFileBytesMock.Expect(ctx, uploadParam).Return(expectedError)

	// Test the method
	err := mockClient.UploadPrivateFileBytes(ctx, uploadParam)

	// Verify results
	qt.Check(err, quicktest.Equals, expectedError)
}

func TestMinioClient_DeleteFile(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"

	// Set up mock expectations
	mockClient.DeleteFileMock.Expect(ctx, userUID, filePath).Return(nil)

	// Test the method
	err := mockClient.DeleteFile(ctx, userUID, filePath)

	// Verify results
	qt.Check(err, quicktest.IsNil)
}

func TestMinioClient_DeleteFile_Error(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"

	// Expected error
	expectedError := errors.New("delete failed")

	// Set up mock expectations
	mockClient.DeleteFileMock.Expect(ctx, userUID, filePath).Return(expectedError)

	// Test the method
	err := mockClient.DeleteFile(ctx, userUID, filePath)

	// Verify results
	qt.Check(err, quicktest.Equals, expectedError)
}

func TestMinioClient_GetFile(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"
	expectedContent := []byte(`{"test": "data"}`)

	// Set up mock expectations
	mockClient.GetFileMock.Expect(ctx, userUID, filePath).Return(expectedContent, nil)

	// Test the method
	content, err := mockClient.GetFile(ctx, userUID, filePath)

	// Verify results
	qt.Check(err, quicktest.IsNil)
	qt.Check(content, quicktest.DeepEquals, expectedContent)
}

func TestMinioClient_GetFile_Error(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePath := "test/file.json"

	// Expected error
	expectedError := errors.New("file not found")

	// Set up mock expectations
	mockClient.GetFileMock.Expect(ctx, userUID, filePath).Return(nil, expectedError)

	// Test the method
	content, err := mockClient.GetFile(ctx, userUID, filePath)

	// Verify results
	qt.Check(err, quicktest.Equals, expectedError)
	qt.Check(content, quicktest.IsNil)
}

func TestMinioClient_GetFilesByPaths(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePaths := []string{"test/file1.json", "test/file2.json"}

	expectedFiles := []miniox.FileContent{
		{
			Name:    "test/file1.json",
			Content: []byte(`{"file1": "data"}`),
		},
		{
			Name:    "test/file2.json",
			Content: []byte(`{"file2": "data"}`),
		},
	}

	// Set up mock expectations
	mockClient.GetFilesByPathsMock.Expect(ctx, userUID, filePaths).Return(expectedFiles, nil)

	// Test the method
	files, err := mockClient.GetFilesByPaths(ctx, userUID, filePaths)

	// Verify results
	qt.Check(err, quicktest.IsNil)
	qt.Check(files, quicktest.DeepEquals, expectedFiles)
}

func TestMinioClient_GetFilesByPaths_Error(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	ctx := context.Background()
	userUID := uuid.Must(uuid.NewV4())
	filePaths := []string{"test/file1.json", "test/file2.json"}

	// Expected error
	expectedError := errors.New("files not found")

	// Set up mock expectations
	mockClient.GetFilesByPathsMock.Expect(ctx, userUID, filePaths).Return(nil, expectedError)

	// Test the method
	files, err := mockClient.GetFilesByPaths(ctx, userUID, filePaths)

	// Verify results
	qt.Check(err, quicktest.Equals, expectedError)
	qt.Check(files, quicktest.IsNil)
}

func TestMinioClient_Client(t *testing.T) {
	qt := quicktest.New(t)
	mc := minimock.NewController(t)

	// Create mock client
	mockClient := mockminio.NewClientMock(mc)

	// Expected MinIO client
	expectedMinioClient := &miniogo.Client{}

	// Set up mock expectations
	mockClient.ClientMock.Expect().Return(expectedMinioClient)

	// Test the method
	client := mockClient.Client()

	// Verify results
	qt.Check(client, quicktest.Equals, expectedMinioClient)
}

// Test helper functions
func TestGenerateInputRefID(t *testing.T) {
	qt := quicktest.New(t)

	prefix := "test-prefix"
	result := miniox.GenerateInputRefID(prefix)

	// Verify the result contains the expected parts
	qt.Check(result, quicktest.Contains, prefix)
	qt.Check(result, quicktest.Contains, "/input/")
	qt.Check(len(result) > len(prefix)+10, quicktest.IsTrue) // Should be longer than prefix + "/input/" + some UUID
}

func TestGenerateOutputRefID(t *testing.T) {
	qt := quicktest.New(t)

	prefix := "test-prefix"
	result := miniox.GenerateOutputRefID(prefix)

	// Verify the result contains the expected parts
	qt.Check(result, quicktest.Contains, prefix)
	qt.Check(result, quicktest.Contains, "/output/")
	qt.Check(len(result) > len(prefix)+11, quicktest.IsTrue) // Should be longer than prefix + "/output/" + some UUID
}

// Integration test for the actual MinIO client (keeping the original test)
func TestMinioIntegration(t *testing.T) {
	qt := quicktest.New(t)

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
	mc, err := miniox.NewMinIOClientAndInitBucket(ctx, params)

	// Skip the test if MinIO is not available
	if err != nil {
		t.Skipf("MinIO not available, skipping integration test: %v", err)
	}

	// Ensure mc is not nil before proceeding
	qt.Check(mc, quicktest.Not(quicktest.IsNil))

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
	qt.Check(err, quicktest.IsNil)
	t.Log("url:", url)
	t.Log("size:", stat.Size)

	fileBytes, err := mc.GetFile(ctx, userUID, fileName.String())
	qt.Check(err, quicktest.IsNil)
	qt.Check(fileBytes, quicktest.DeepEquals, jsonBytes)

	err = mc.DeleteFile(ctx, userUID, fileName.String())
	qt.Check(err, quicktest.IsNil)

	_, err = mc.GetFile(ctx, userUID, fileName.String())
	qt.Check(err, quicktest.Not(quicktest.IsNil))
}

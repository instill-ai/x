package repo_test

import (
	"fmt"
	"testing"

	"github.com/instill-ai/x/repo"
	"github.com/stretchr/testify/require"
)

func TestReadReleaseManifest_NoError(t *testing.T) {
	manifestFilePath := "../release-please/manifest.json"
	v, err := repo.ReadReleaseManifest(manifestFilePath)
	require.NoError(t, err)
	require.NotEmpty(t, v)
}

func TestReadReleaseManifest_Error(t *testing.T) {
	manifestFilePath := "non-exist-manifest.json"
	v, err := repo.ReadReleaseManifest(manifestFilePath)
	require.EqualError(t, err, fmt.Sprintf("open %v: no such file or directory", manifestFilePath))
	require.Empty(t, v)
}

func TestReadReleaseManifest_Invalid(t *testing.T) {
	manifestFilePath := "../release-please/config.json"
	v, err := repo.ReadReleaseManifest(manifestFilePath)
	require.EqualError(t, err, "invalid release manifest file")
	require.Empty(t, v)
}

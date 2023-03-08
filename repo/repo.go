package repo

import (
	"encoding/json"
	"errors"
	"os"
)

// ReadReleaseManifest reads a repo's `release-please/manifest.json` file
// and returns the release version
// format of the manifest.json
// {
// 	".": "release_version"
// }
func ReadReleaseManifest(manifestFilepath string) (string, error) {
	type Release struct {
		Version string `json:".,"` // field appears in JSON as key "."
	}

	content, err := os.ReadFile(manifestFilepath)
	if err != nil {
		return "", err
	}
	release := Release{}
	err = json.Unmarshal(content, &release)
	if err != nil {
		return "", err
	}
	if release.Version == "" {
		return "", errors.New("invalid release manifest file")
	}
	return release.Version, nil
}

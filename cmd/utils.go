package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Networking

func Fetch(input string) ([]byte, error) {
	resp, err := http.Get(input)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func removeScope(pkgName string) string {
	if strings.HasPrefix(pkgName, "@") {
		return strings.Split(pkgName, "/")[1]
	}
	return pkgName
}

// Archive

func ReadFileFromArchive(fileName string, data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	gr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gr)
	for {
		th, err := tr.Next()
		if err != nil {
			break
		}

		if th.FileInfo().Name() == fileName {
			return ioutil.ReadAll(tr)
		}
	}

	return nil, errors.New("file not found")
}

func ReadPackageJsonFromArchive(data []byte) ([]byte, error) {
	return ReadFileFromArchive("package.json", data)
}

// Semver

func max(v1, v2 *semver.Version) *semver.Version {
	if v1.GreaterThan(v2) || v1.Equal(v2) {
		return v1
	}

	return v2
}

func MaxSatisfying(reference string, versions map[string]interface{}) (string, error) {
	c, err := semver.NewConstraint(reference)
	if err != nil {
		return "", err
	}

	var maxSatisfying *semver.Version
	for version := range versions {
		v, err := semver.NewVersion(version)
		if err != nil {
			return "", err
		}
		if c.Check(v) {
			if maxSatisfying == nil {
				maxSatisfying = v
				continue
			}
			maxSatisfying = max(maxSatisfying, v)
		}
	}

	if maxSatisfying != nil {
		return maxSatisfying.String(), nil
	}
	return "", errors.New("none of versions satisfy the constraint")
}

func checkParentSatisfies(parentReference, childReference string) bool {
	c, err := semver.NewConstraint(childReference)
	if err != nil {
		return false
	}
	// parentReference must be valid
	v, _ := semver.NewVersion(parentReference)
	return c.Check(v)
}

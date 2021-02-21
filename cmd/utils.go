package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// package.json

type PackageJson struct {
	Bin interface{} `json:"bin"`
	Scripts map[string]string `json:"scripts"`
	Dependencies map[string]string `json:"dependencies"`
}

func ReadPackageJson(data []byte, target *PackageJson) error {
	return json.Unmarshal(data, target)
}

func ReadPackageJsonFromDisk(path string, target *PackageJson) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return ReadPackageJson(content, target)
}

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

func ExtractArchiveTo(data []byte, target string) error {
	buf := bytes.NewBuffer(data)
	gr, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)
	for {
		th, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case th == nil:
			continue
		}

		dest := filepath.Join(
			target,
			strings.Replace(th.Name, "package/", "", 1),
		)
		switch th.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				if err := os.MkdirAll(dest, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				if err := os.MkdirAll(path.Dir(dest), 0755); err != nil {
					return err
				}
			}

			f, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR, os.FileMode(th.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}

func ExtractNpmArchiveTo(data []byte, target string) error {
	return ExtractArchiveTo(data, target)
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

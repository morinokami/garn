package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

func FetchPackage(name, reference string) []byte {
	_, err := semver.NewVersion(reference)
	if err != nil {
		resp, err := Fetch(reference)
		if err != nil {
			panic(err)
		}
		return resp
	}

	return FetchPackage(name, fmt.Sprintf("https://registry.yarnpkg.com/%s/-/%s-%s.tgz", name, name, reference))
}

type PackageInfo struct {
	Versions map[string]interface{} `json:"versions"`
}

func GetPinnedReference(name, reference string) (string, string) {
	_, err := semver.NewVersion(reference)
	if err != nil {
		// Download package info
		res, err := Fetch(fmt.Sprintf("https://registry.yarnpkg.com/%s", name))
		if err != nil {
			panic(err)
		}
		var info PackageInfo
		err = json.Unmarshal(res, &info)
		if err != nil {
			panic(err)
		}

		// Search for maxSatisfying
		maxSatisfying, err := MaxSatisfying(reference, info.Versions)
		if err != nil {
			panic(err)
		}

		return name, maxSatisfying
	}

	return name, reference
}

type PackageJson struct {
	Dependencies map[string]string `json:"dependencies"`
}

type Dependency struct {
	Name      string
	Reference string
}

func GetPackageDependencies(name, reference string) []Dependency {
	packageBuffer := FetchPackage(name, reference)
	pkgData, err := ReadPackageJsonFromArchive(packageBuffer)
	if err != nil {
		panic(err)
	}

	var packageJson PackageJson
	err = json.Unmarshal(pkgData, &packageJson)
	if err != nil {
		panic(err)
	}

	var dependencies []Dependency
	for dep := range packageJson.Dependencies {
		dependencies = append(dependencies, Dependency{dep, packageJson.Dependencies[name]})
	}

	return dependencies
}

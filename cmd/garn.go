package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

func FetchPackage(pkg Package) []byte {
	_, err := semver.NewVersion(pkg.Reference)
	if err != nil {
		resp, err := Fetch(pkg.Reference)
		if err != nil {
			panic(err)
		}
		return resp
	}

	return FetchPackage(
		Package{
			pkg.Name,
			fmt.Sprintf(
				"https://registry.yarnpkg.com/%s/-/%s-%s.tgz",
				pkg.Name,
				removeScope(pkg.Name),
				pkg.Reference,
			),
		},
	)
}

type PackageInfo struct {
	Versions map[string]interface{} `json:"versions"`
}

func GetPinnedReference(pkg Package) Package {
	v, err := semver.NewVersion(pkg.Reference)
	if err != nil {
		// Download package info
		res, err := Fetch(fmt.Sprintf("https://registry.yarnpkg.com/%s", pkg.Name))
		if err != nil {
			panic(err)
		}
		var info PackageInfo
		err = json.Unmarshal(res, &info)
		if err != nil {
			panic(err)
		}

		// Search for maxSatisfying
		maxSatisfying, err := MaxSatisfying(pkg.Reference, info.Versions)
		if err != nil {
			panic(err)
		}

		return Package{pkg.Name, maxSatisfying}
	}

	return Package{pkg.Name, v.String()}
}

type PackageJson struct {
	Dependencies map[string]string `json:"dependencies"`
}

type Package struct {
	Name      string
	Reference string
}

func GetPackageDependencies(pkg Package) []Package {
	packageBuffer := FetchPackage(pkg)
	pkgData, err := ReadPackageJsonFromArchive(packageBuffer)
	if err != nil {
		panic(err)
	}

	var packageJson PackageJson
	err = json.Unmarshal(pkgData, &packageJson)
	if err != nil {
		panic(err)
	}

	var dependencies []Package
	for dep := range packageJson.Dependencies {
		dependencies = append(dependencies, Package{dep, packageJson.Dependencies[dep]})
	}

	return dependencies
}

type DependencyNode struct {
	Name         string
	Reference    string
	Dependencies []DependencyNode
}

func GetPackageDependencyTree(pkg Package, dependencies []Package, available map[string]string) DependencyNode {
	dependencyTree := DependencyNode{Name: pkg.Name, Reference: pkg.Reference}
	for _, volatileDependency := range dependencies {
		if availableReference, ok := available[volatileDependency.Name]; ok {
			if volatileDependency.Reference == availableReference ||
				checkParentSatisfies(availableReference, volatileDependency.Reference) {
				continue
			}
		}

		pinnedDependency := GetPinnedReference(volatileDependency)
		subDependencies := GetPackageDependencies(pinnedDependency)

		subAvailable := make(map[string]string)
		for name, reference := range available {
			subAvailable[name] = reference
		}
		subAvailable[pinnedDependency.Name] = pinnedDependency.Reference

		dependencyTree.Dependencies = append(
			dependencyTree.Dependencies,
			GetPackageDependencyTree(pinnedDependency, subDependencies, subAvailable),
		)
	}

	return dependencyTree
}

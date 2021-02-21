package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"

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
	err = ReadPackageJson(pkgData, &packageJson)
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
	fmt.Println("Getting dependencies for", pkg.Name)
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

func LinkPackages(node DependencyNode, cwd string) {
	if len(node.Reference) > 0 {
		packageBuffer := FetchPackage(Package{node.Name, node.Reference})
		err := ExtractNpmArchiveTo(packageBuffer, cwd)
		if err != nil {
			panic(err)
		}
	}

	for _, dependency := range node.Dependencies {
		target := fmt.Sprintf("%s/node_modules/%s", cwd, dependency.Name)
		binTarget := fmt.Sprintf("%s/node_modules/.bin", cwd)

		// linking
		LinkPackages(dependency, target)

		// bin
		var dependencyPackageJson PackageJson
		err := ReadPackageJsonFromDisk(
			fmt.Sprintf("%s/package.json", target),
			&dependencyPackageJson,
		)
		if err != nil {
			panic(err)
		}

		var bin map[string]interface{}
		switch reflect.ValueOf(dependencyPackageJson.Bin).Kind() {
		case reflect.String:
			bin = map[string]interface{}{
				dependency.Name: dependencyPackageJson.Bin.(string),
			}
		case reflect.Map:
			bin = dependencyPackageJson.Bin.(map[string]interface{})
		}

		for binName := range bin {
			binPath := bin[binName].(string)
			source := path.Join("..", dependency.Name, binPath)
			dest := fmt.Sprintf("%s/%s", binTarget, binName)

			err := os.MkdirAll(fmt.Sprintf("%s/node_modules/.bin", cwd), 0755)
			if err != nil {
				panic(err)
			}
			err = os.Symlink(source, dest)
			if err != nil {
				panic(err)
			}
		}

		// TODO: scripts (working now?)
		if dependencyPackageJson.Scripts != nil {
			for _, scriptName := range []string{"preinstall", "install", "postinstall"} {
				if script, ok := dependencyPackageJson.Scripts[scriptName]; ok {
					fmt.Printf(
						"Running %s script for %s: %s\n",
						scriptName,
						dependencyPackageJson.Name,
						script,
					)
					splitScript := strings.Split(script, " ")
					cmd := exec.Command(splitScript[0], splitScript[1:]...)
					cmd.Dir = target
					cmd.Env = append(
						os.Environ(),
						fmt.Sprintf("PATH=%s/node_modules/.bin:%s", target, os.Getenv("PATH")),
					)
					out, err := cmd.Output()
					if err != nil {
						panic(err)
					}
					fmt.Println("Output:", string(out))
				}
			}
		}
	}
}

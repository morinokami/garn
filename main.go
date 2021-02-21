package main

import (
	"fmt"
	"github.com/morinokami/garn/cmd"
)

func main() {
	var packageJson cmd.PackageJson
	err := cmd.ReadPackageJsonFromDisk("./package.json", &packageJson)
	if err != nil {
		panic(err)
	}
	pkg := cmd.Package{
		Name: packageJson.Name,
	}
	var dependencies []cmd.Package
	for name, reference := range packageJson.Dependencies {
		dependencies = append(dependencies, cmd.Package{
			Name:      name,
			Reference: reference,
		})
	}
	available := make(map[string]string)

	fmt.Println("Resolving packages...")
	tree := cmd.GetPackageDependencyTree(pkg, dependencies, available)

	fmt.Println("Linking dependencies...")
	cmd.LinkPackages(tree, ".")
}

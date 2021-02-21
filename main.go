package main

import (
	"github.com/morinokami/garn/cmd"
)

func main() {
	pkg := cmd.Package{Name: "my-awesome-package"}
	dependencies := []cmd.Package{
		{
			Name:      "react",
			Reference: "^16.10.0",
		},
	}
	available := make(map[string]string)

	tree := cmd.GetPackageDependencyTree(pkg, dependencies, available)
	cmd.LinkPackages(tree, "/home/shf0811/dev/garn")
}

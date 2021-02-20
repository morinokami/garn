package main

import (
	"github.com/morinokami/garn/cmd"
)

func main() {
	pkg := cmd.Package{Name: "my-awesome-package"}
	dependencies := []cmd.Package{
		{
			Name:      "eslint",
			Reference: "^7.6.0",
		},
	}
	available := make(map[string]string)

	cmd.GetPackageDependencyTree(pkg, dependencies, available)
}

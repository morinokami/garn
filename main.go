package main

import (
	"fmt"

	"github.com/morinokami/garn/cmd"
)

func main() {
	fmt.Println(cmd.GetPinnedReference("react", "^16.10.0")) // 16.14.0
	fmt.Println(cmd.GetPackageDependencies("react", "16.14.0"))
}

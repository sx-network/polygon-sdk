package main

import (
	_ "embed"
	"fmt"

	"github.com/0xPolygon/polygon-edge/command/root"
	"github.com/0xPolygon/polygon-edge/licenses"
)

var (
	//go:embed LICENSE
	license string
)

func main() {
	fmt.Println("[main] 1")
	licenses.SetLicense(license)

	root.NewRootCommand().Execute()
	fmt.Println("[main] 2")
}

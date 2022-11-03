package main

import (
	_ "embed"
	_ "net/http/pprof"

	"github.com/0xPolygon/polygon-edge/command/root"
	"github.com/0xPolygon/polygon-edge/licenses"
)

var (
	//go:embed LICENSE
	license string
)

func main() {
	licenses.SetLicense(license)

	root.NewRootCommand().Execute()
}

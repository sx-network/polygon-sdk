package root

import (
	"fmt"
	"os"

	"github.com/0xPolygon/polygon-edge/command/backup"
	"github.com/0xPolygon/polygon-edge/command/datafeed"
	"github.com/0xPolygon/polygon-edge/command/genesis"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/command/ibft"
	"github.com/0xPolygon/polygon-edge/command/license"
	"github.com/0xPolygon/polygon-edge/command/loadbot"
	"github.com/0xPolygon/polygon-edge/command/monitor"
	"github.com/0xPolygon/polygon-edge/command/peers"
	"github.com/0xPolygon/polygon-edge/command/secrets"
	"github.com/0xPolygon/polygon-edge/command/server"
	"github.com/0xPolygon/polygon-edge/command/status"
	"github.com/0xPolygon/polygon-edge/command/txpool"
	"github.com/0xPolygon/polygon-edge/command/whitelist"
	"github.com/spf13/cobra"
)

type RootCommand struct {
	baseCmd *cobra.Command
}

func NewRootCommand() *RootCommand {
	fmt.Println("[root] 1")
	rootCommand := &RootCommand{
		baseCmd: &cobra.Command{
			Short: "Polygon Edge is a framework for building Ethereum-compatible Blockchain networks",
		},
	}
	fmt.Println("[root] 2")
	helper.RegisterJSONOutputFlag(rootCommand.baseCmd)
	fmt.Println("[root] 3")
	rootCommand.registerSubCommands()
	fmt.Println("[root] 4")
	return rootCommand
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		// version.GetCommand(),
		txpool.GetCommand(),
		status.GetCommand(),
		secrets.GetCommand(),
		peers.GetCommand(),
		monitor.GetCommand(),
		loadbot.GetCommand(),
		ibft.GetCommand(),
		backup.GetCommand(),
		genesis.GetCommand(),
		server.GetCommand(),
		whitelist.GetCommand(),
		license.GetCommand(),
		datafeed.GetCommand(),
	)
}

func (rc *RootCommand) Execute() {
	fmt.Println("[root] 5")
	if err := rc.baseCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
	fmt.Println("[root] 6")
}

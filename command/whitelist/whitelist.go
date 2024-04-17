package whitelist

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/command/whitelist/deployment"
	"github.com/0xPolygon/polygon-edge/command/whitelist/show"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	fmt.Println("[whitelist] 1")
	whitelistCmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Top level command for modifying the Polygon Edge whitelists within the config. Only accepts subcommands.",
	}

	registerSubcommands(whitelistCmd)

	return whitelistCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		deployment.GetCommand(),
		show.GetCommand(),
	)
}

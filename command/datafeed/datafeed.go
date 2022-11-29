package datafeed

import (
	report "github.com/0xPolygon/polygon-edge/command/datafeed/reportoutcome"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	dataFeedCmd := &cobra.Command{
		Use:   "datafeed",
		Short: "Top level command for interacting with the datafeed service. Only accepts subcommands.",
	}

	helper.RegisterGRPCAddressFlag(dataFeedCmd)

	registerSubcommands(dataFeedCmd)

	return dataFeedCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// datafeed report outcome
		report.GetCommand(),
	)
}

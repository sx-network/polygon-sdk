package reportoutcome

import (
	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/spf13/cobra"
)

const (
	marketFlag  = "market"
	outcomeFlag = "outcome"
)

func GetCommand() *cobra.Command {
	reportOutcomeCmd := &cobra.Command{
		Use:   "report",
		Short: "Reports a new market outcome",
		Run:   runCommand,
	}

	setFlags(reportOutcomeCmd)
	setRequiredFlags(reportOutcomeCmd)

	return reportOutcomeCmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.market,
		marketFlag,
		"",
		"the market hash of the report",
	)

	cmd.Flags().StringVar(
		&params.outcome,
		outcomeFlag,
		"",
		"the outcome of the report",
	)
}

func setRequiredFlags(cmd *cobra.Command) {
	for _, requiredFlag := range params.getRequiredFlags() {
		_ = cmd.MarkFlagRequired(requiredFlag)
	}
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	if err := params.reportOutcome(helper.GetGRPCAddress(cmd)); err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(params.getResult())
}

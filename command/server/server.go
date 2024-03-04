package server

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/helper"
	"github.com/0xPolygon/polygon-edge/command/server/config"
	"github.com/0xPolygon/polygon-edge/command/server/export"
	"github.com/0xPolygon/polygon-edge/server"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:     "server",
		Short:   "The default command that starts the Polygon Edge client, by bootstrapping all modules together",
		PreRunE: runPreRun,
		Run:     runCommand,
	}

	helper.RegisterGRPCAddressFlag(serverCmd)
	helper.RegisterLegacyGRPCAddressFlag(serverCmd)
	helper.RegisterJSONRPCFlag(serverCmd)

	registerSubcommands(serverCmd)
	setFlags(serverCmd)

	return serverCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// server export
		export.GetCommand(),
	)
}

func setFlags(cmd *cobra.Command) {
	defaultConfig := config.DefaultConfig()

	cmd.Flags().StringVar(
		&params.rawConfig.LogLevel,
		command.LogLevelFlag,
		defaultConfig.LogLevel,
		"the log level for console output",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.GenesisPath,
		genesisPathFlag,
		defaultConfig.GenesisPath,
		"the genesis file used for starting the chain",
	)

	cmd.Flags().StringVar(
		&params.configPath,
		configFlag,
		"",
		"the path to the CLI config. Supports .json and .hcl",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataDir,
		dataDirFlag,
		defaultConfig.DataDir,
		"the data directory used for storing Polygon Edge client data",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.Network.Libp2pAddr,
		libp2pAddressFlag,
		defaultConfig.Network.Libp2pAddr,
		"the address and port for the libp2p service",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.Telemetry.PrometheusAddr,
		prometheusAddressFlag,
		"",
		"the address and port for the prometheus instrumentation service (address:port). "+
			"If only port is defined (:port) it will bind to 0.0.0.0:port",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.Network.NatAddr,
		natFlag,
		"",
		"the external IP address without port, as can be seen by peers",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.Network.DNSAddr,
		dnsFlag,
		"",
		"the host DNS address which can be used by a remote peer for connection",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.BlockGasTarget,
		blockGasTargetFlag,
		defaultConfig.BlockGasTarget,
		"the target block gas limit for the chain. If omitted, the value of the parent block is used",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.SecretsConfigPath,
		secretsConfigFlag,
		"",
		"the path to the SecretsManager config file. Used for Hashicorp Vault. "+
			"If omitted, the local FS secrets manager is used",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.RestoreFile,
		restoreFlag,
		"",
		"the path to the archive blockchain data to restore on initialization",
	)

	cmd.Flags().BoolVar(
		&params.rawConfig.ShouldSeal,
		sealFlag,
		defaultConfig.ShouldSeal,
		"the flag indicating that the client should seal blocks",
	)

	cmd.Flags().BoolVar(
		&params.rawConfig.Network.NoDiscover,
		command.NoDiscoverFlag,
		defaultConfig.Network.NoDiscover,
		"prevent the client from discovering other peers",
	)

	cmd.Flags().BoolVar(
		&params.rawConfig.Network.BootnodeOnlySync,
		command.BootnodeOnlySyncFlag,
		defaultConfig.Network.BootnodeOnlySync,
		"restrict the client to only sync to bootnodes (default: false)",
	)

	cmd.Flags().Int64Var(
		&params.rawConfig.Network.MaxPeers,
		maxPeersFlag,
		-1,
		"the client's max number of peers allowed",
	)
	// override default usage value
	cmd.Flag(maxPeersFlag).DefValue = fmt.Sprintf("%d", defaultConfig.Network.MaxPeers)

	cmd.Flags().Int64Var(
		&params.rawConfig.Network.MaxInboundPeers,
		maxInboundPeersFlag,
		-1,
		"the client's max number of inbound peers allowed",
	)
	// override default usage value
	cmd.Flag(maxInboundPeersFlag).DefValue = fmt.Sprintf("%d", defaultConfig.Network.MaxInboundPeers)
	cmd.MarkFlagsMutuallyExclusive(maxPeersFlag, maxInboundPeersFlag)

	cmd.Flags().Int64Var(
		&params.rawConfig.Network.MaxOutboundPeers,
		maxOutboundPeersFlag,
		-1,
		"the client's max number of outbound peers allowed",
	)
	// override default usage value
	cmd.Flag(maxOutboundPeersFlag).DefValue = fmt.Sprintf("%d", defaultConfig.Network.MaxOutboundPeers)
	cmd.MarkFlagsMutuallyExclusive(maxPeersFlag, maxOutboundPeersFlag)

	cmd.Flags().Uint64Var(
		&params.rawConfig.TxPool.PriceLimit,
		priceLimitFlag,
		defaultConfig.TxPool.PriceLimit,
		fmt.Sprintf(
			"the minimum gas price limit to enforce for acceptance into the pool (default %d)",
			defaultConfig.TxPool.PriceLimit,
		),
	)

	cmd.Flags().Uint64Var(
		&params.rawConfig.TxPool.MaxSlots,
		maxSlotsFlag,
		defaultConfig.TxPool.MaxSlots,
		"maximum slots in the pool",
	)

	cmd.Flags().Uint64Var(
		&params.rawConfig.TxPool.MaxAccountEnqueued,
		maxEnqueuedFlag,
		defaultConfig.TxPool.MaxAccountEnqueued,
		"maximum number of enqueued transactions per account",
	)

	cmd.Flags().Uint64Var(
		&params.rawConfig.BlockTime,
		blockTimeFlag,
		defaultConfig.BlockTime,
		"minimum block time in seconds (at least 1s)",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.RPCNrAppName,
		rpcNRAppNameFlag,
		"",
		"the New Relic Agent App Name, used for reporting json-rpc metrics.",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.RPCNrLicenseKey,
		rpcNRLicenseKeyFlag,
		"",
		"the New Relic Agent License Key, used for reporting json-rpc metrics.",
	)

	cmd.Flags().Float64Var(
		&params.rawConfig.GasPriceBlockUtilizationMinimum,
		gasPriceBlockUtilizationMinimumFlag,
		0,
		"the gas price utilization threshold used to compute eth_gasPrice."+
			"If a block's gas utilization is under this threshold, it will not affect the running average gas price.",
	)

	cmd.Flags().StringArrayVar(
		&params.corsAllowedOrigins,
		corsOriginFlag,
		defaultConfig.Headers.AccessControlAllowOrigins,
		"the CORS header indicating whether any JSON-RPC response can be shared with the specified origin",
	)

	cmd.Flags().Uint64Var(
		&params.rawConfig.JSONRPCBatchRequestLimit,
		jsonRPCBatchRequestLimitFlag,
		defaultConfig.JSONRPCBatchRequestLimit,
		"max length to be considered when handling json-rpc batch requests, value of 0 disables it",
	)

	cmd.Flags().Uint64Var(
		&params.rawConfig.JSONRPCBlockRangeLimit,
		jsonRPCBlockRangeLimitFlag,
		defaultConfig.JSONRPCBlockRangeLimit,
		"max block range to be considered when executing json-rpc requests "+
			"that consider fromBlock/toBlock values (e.g. eth_getLogs), value of 0 disables it",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.LogFilePath,
		logFileLocationFlag,
		defaultConfig.LogFilePath,
		"write all logs to the file at specified location instead of writing them to console",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.AMQPURI,
		dataFeedAMQPURIFlag,
		"",
		"the AMQP URI used by the DataFeedService's MQConsumer",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.AMQPExchangeName,
		dataFeedAMQPExchangeNameFlag,
		"",
		"the AMQ exchange that the DataFeedService's MQConsumer consumes from",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.AMQPQueueName,
		dataFeedAMQPQueueNameFlag,
		"",
		"the AMQ queue that the DataFeedService's MQConsumer consumes from",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.VerifyOutcomeAPIURL,
		verifyOutcomeAPIURLFlag,
		defaultConfig.DataFeed.VerifyOutcomeAPIURL,
		"the GET URL to verify market outcome, used by the DataFeedService",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.OutcomeReporterAddress,
		outcomeReporterAddressFlag,
		defaultConfig.DataFeed.OutcomeReporterAddress,
		"the address of the OutcomeReporter contract, used by the DataFeedService's EventListener",
	)

	cmd.Flags().StringVar(
		&params.rawConfig.DataFeed.SXNodeAddress,
		sxNodeAddressFlag,
		defaultConfig.DataFeed.SXNodeAddress,
		"the address of the SXNode contract, used by DataFeedService reporting txs",
	)

	cmd.Flags().BoolVar(
		&params.isEnable,
		isEnableFlag,
		defaultConfig.Monitoring.IsEnable,
		"indicates when pprof should start the snapshot profile routine",
	)

	cmd.Flags().Uint64Var(
		&params.delayInSeconds,
		delayInSecondsFlag,
		defaultConfig.Monitoring.DelayInSeconds,
		"indicates the delay time the prof will get from the snapshot profile",
	)

	cmd.Flags().BoolVar(
		&params.isMemStressTestEnable,
		isMemStressTestEnableFlag,
		defaultConfig.Monitoring.IsMemStressTestEnable,
		"flag that triggers the memory stress test function",
	)

	cmd.Flags().Uint64Var(
		&params.tickerInSeconds,
		tickerInSecondsFlag,
		defaultConfig.Monitoring.TickerInSeconds,
		"indicates the delay time that the memory stress test function will allocate memory",
	)

	cmd.Flags().Float64Var(
		&params.threshold,
		thresholdFlag,
		defaultConfig.Monitoring.Threshold,
		"the threshold for memory usage, specified as a percentage between 0 and 1",
	)

	setLegacyFlags(cmd)

	setDevFlags(cmd)
}

// setLegacyFlags sets the legacy flags to preserve backwards compatibility
// with running partners
func setLegacyFlags(cmd *cobra.Command) {
	// Legacy IBFT base timeout flag
	cmd.Flags().Uint64Var(
		&params.ibftBaseTimeoutLegacy,
		ibftBaseTimeoutFlagLEGACY,
		0,
		"",
	)

	_ = cmd.Flags().MarkHidden(ibftBaseTimeoutFlagLEGACY)
}

func setDevFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(
		&params.isDevMode,
		devFlag,
		false,
		"should the client start in dev mode (default false)",
	)

	_ = cmd.Flags().MarkHidden(devFlag)

	cmd.Flags().Uint64Var(
		&params.devInterval,
		devIntervalFlag,
		0,
		"the client's dev notification interval in seconds (default 1)",
	)

	_ = cmd.Flags().MarkHidden(devIntervalFlag)
}

func runPreRun(cmd *cobra.Command, _ []string) error {
	// Set the grpc and json ip:port bindings
	// The config file will have precedence over --flag
	params.setRawGRPCAddress(helper.GetGRPCAddress(cmd))
	params.setRawJSONRPCAddress(helper.GetJSONRPCAddress(cmd))
	params.setJSONLogFormat(helper.GetJSONLogFormat(cmd))

	// Check if the config file has been specified
	// Config file settings will override JSON-RPC and GRPC address values
	if isConfigFileSpecified(cmd) {
		if err := params.initConfigFromFile(); err != nil {
			return err
		}
	}

	if err := params.initRawParams(); err != nil {
		return err
	}

	return nil
}

func isConfigFileSpecified(cmd *cobra.Command) bool {
	return cmd.Flags().Changed(configFlag)
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)

	if err := runServerLoop(params.generateConfig(), outputter); err != nil {
		outputter.SetError(err)
		outputter.WriteOutput()

		return
	}
}

func runServerLoop(
	config *server.Config,
	outputter command.OutputFormatter,
) error {
	serverInstance, err := server.NewServer(config)
	if err != nil {
		return err
	}

	return helper.HandleSignals(serverInstance.Close, outputter)
}

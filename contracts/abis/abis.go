package abis

import (
	"github.com/umbracle/ethgo/abi"
)

var OutcomeReporterABI = abi.MustNewABI(OutcomeReporterJSONABI)
var StakingABI = abi.MustNewABI(StakingJSONABI)
var StressTestABI = abi.MustNewABI(StressTestJSONABI)

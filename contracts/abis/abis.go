package abis

import (
	"github.com/umbracle/ethgo/abi"
)

var SXNodeABI = abi.MustNewABI(SXNodeJSONABI)
var StakingABI = abi.MustNewABI(StakingJSONABI)
var StressTestABI = abi.MustNewABI(StressTestJSONABI)

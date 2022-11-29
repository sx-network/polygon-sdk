package abis

import (
	"github.com/umbracle/ethgo/abi"
)

var (
	// ABI for Staking Contract
	StakingABI = abi.MustNewABI(StakingJSONABI)

	// ABI for Contract used in e2e stress test
	StressTestABI = abi.MustNewABI(StressTestJSONABI)

	// ABI for upgradeable SXNode contract to interface with various features
	SXNodeABI = abi.MustNewABI(SXNodeJSONABI)
)

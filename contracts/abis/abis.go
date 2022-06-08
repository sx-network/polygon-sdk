package abis

import (
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/go-web3/abi"
)

var StakingABI = abi.MustNewABI(StakingJSONABI)
var StressTestABI = abi.MustNewABI(StressTestJSONABI)
var USDCContractABI = ethgoabi.MustNewABI(USDCABI)

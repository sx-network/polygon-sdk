package chain

import (
	"math/big"
)

// Params are all the set of params for the chain
type Params struct {
	Forks          *Forks                 `json:"forks"`
	ChainID        int                    `json:"chainID"`
	Engine         map[string]interface{} `json:"engine"`
	BlockGasTarget uint64                 `json:"blockGasTarget"`
	FaultyMode		 FaultyMode							`json:"faultyMode"`
}

func (p *Params) GetEngine() string {
	// We know there is already one
	for k := range p.Engine {
		return k
	}
	return ""
}

// Forks specifies when each fork is activated
type Forks struct {
	Homestead      *Fork `json:"homestead,omitempty"`
	Byzantium      *Fork `json:"byzantium,omitempty"`
	Constantinople *Fork `json:"constantinople,omitempty"`
	Petersburg     *Fork `json:"petersburg,omitempty"`
	Istanbul       *Fork `json:"istanbul,omitempty"`
	EIP150         *Fork `json:"EIP150,omitempty"`
	EIP158         *Fork `json:"EIP158,omitempty"`
	EIP155         *Fork `json:"EIP155,omitempty"`
}

func (f *Forks) active(ff *Fork, block uint64) bool {
	if ff == nil {
		return false
	}
	return ff.Active(block)
}

func (f *Forks) IsHomestead(block uint64) bool {
	return f.active(f.Homestead, block)
}

func (f *Forks) IsByzantium(block uint64) bool {
	return f.active(f.Byzantium, block)
}

func (f *Forks) IsConstantinople(block uint64) bool {
	return f.active(f.Constantinople, block)
}

func (f *Forks) IsPetersburg(block uint64) bool {
	return f.active(f.Petersburg, block)
}

func (f *Forks) IsEIP150(block uint64) bool {
	return f.active(f.EIP150, block)
}

func (f *Forks) IsEIP158(block uint64) bool {
	return f.active(f.EIP158, block)
}

func (f *Forks) IsEIP155(block uint64) bool {
	return f.active(f.EIP155, block)
}

func (f *Forks) At(block uint64) ForksInTime {
	return ForksInTime{
		Homestead:      f.active(f.Homestead, block),
		Byzantium:      f.active(f.Byzantium, block),
		Constantinople: f.active(f.Constantinople, block),
		Petersburg:     f.active(f.Petersburg, block),
		Istanbul:       f.active(f.Istanbul, block),
		EIP150:         f.active(f.EIP150, block),
		EIP158:         f.active(f.EIP158, block),
		EIP155:         f.active(f.EIP155, block),
	}
}

type Fork uint64

func NewFork(n uint64) *Fork {
	f := Fork(n)

	return &f
}

func (f Fork) Active(block uint64) bool {
	return block >= uint64(f)
}

func (f Fork) Int() *big.Int {
	return big.NewInt(int64(f))
}

type ForksInTime struct {
	Homestead,
	Byzantium,
	Constantinople,
	Petersburg,
	Istanbul,
	EIP150,
	EIP158,
	EIP155 bool
}

var AllForksEnabled = &Forks{
	Homestead:      NewFork(0),
	EIP150:         NewFork(0),
	EIP155:         NewFork(0),
	EIP158:         NewFork(0),
	Byzantium:      NewFork(0),
	Constantinople: NewFork(0),
	Petersburg:     NewFork(0),
	Istanbul:       NewFork(0),
}

type FaultyMode uint64

const (
	// Disabled disables the faulty mode
	Disabled FaultyMode = iota
	// Random attacks randomly
	Random

	// NotGossiped doesn't gossip any messages to other validators
	NotGossiped
	// SendWrongMsg sends the message with a randomly-generated type
	SendWrongMsgType
	// SendWrongMsgSeal sends the message with a randomly-generated seal
	SendWrongMsgSeal
	// SendWrongMsgSignature sends the message with a randomly-generated signature
	SendWrongMsgSignature
	// SendWrongMsgView sends the message with a randomly-generated view
	SendWrongMsgView
	// SendWrongMsgDigest sends the message with a randomly-generated digest
	SendWrongMsgDigest
	// SendWrongMsgProposal sends the message with a randomly-generated proposal
	SendWrongMsgProposal
	
	// AlwaysPropose always proposes a proposal to validators
	AlwaysPropose
	// AlwaysRoundChange always sends round change while receiving messages
	AlwaysRoundChange
	// BadBlock always proposes a block with bad body
	BadBlock
)

func (f FaultyMode) Uint64() uint64 {
	return uint64(f)
}

func (f FaultyMode) String() string {
	switch f {
	case Disabled:
		return "Disabled"
	case Random:
		return "Random"

	case NotGossiped:
		return "NotGossiped"
	case SendWrongMsgType:
		return "SendWrongMsgType"
	case SendWrongMsgSeal:
		return "SendWrongMsgSeal"
	case SendWrongMsgSignature:
		return "SendWrongMsgSignature"
	case SendWrongMsgView:
		return "SendWrongMsgView"
	case SendWrongMsgDigest:
		return "SendWrongMsgDigest"
	case SendWrongMsgProposal:
		return "SendWrongMsgProposal"

	case AlwaysPropose:
		return "AlwaysPropose"
	case AlwaysRoundChange:
		return "AlwaysRoundChange"
	case BadBlock:
		return "BadBlock"
	default:
		return "Undefined"
	}
}

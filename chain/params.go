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
	FaultyMode		 FaultyModeValue
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

type FaultyModeValue struct {
	Value uint64
}

const (
	// Disabled disables the faulty mode
	Disabled = 0
	// Random attacks randomly
	Random = 1

	// NotGossiped doesn't gossip any messages to other validators
	NotGossiped = 2
	// SendWrongMsg sends the message with a randomly-generated type
	SendWrongMsgType = 3
	// SendWrongMsgSeal sends the message with a randomly-generated seal
	SendWrongMsgSeal = 4
	// SendWrongMsgSignature sends the message with a randomly-generated signature
	SendWrongMsgSignature = 5
	// SendWrongMsgView sends the message with a randomly-generated view
	SendWrongMsgView = 6
	// SendWrongMsgDigest sends the message with a randomly-generated digest
	SendWrongMsgDigest = 7
	// SendWrongMsgProposal sends the message with a randomly-generated proposal
	SendWrongMsgProposal = 8
	
	// AlwaysPropose always proposes a proposal to validators
	AlwaysPropose = 9
	// AlwaysRoundChange always sends round change while receiving messages
	AlwaysRoundChange = 10
	// BadBlock always proposes a block with bad body
	BadBlock = 11
)

func (f FaultyModeValue) SetFaultyMode(value uint64) {
	f.Value = value
}
func (f FaultyModeValue) GetFaultyMode() uint64 {
	return f.Value
}

func (f FaultyModeValue) IsDisabled() bool {
	return f.Value == Disabled
}

func (f FaultyModeValue) IsRandom() bool {
	return f.Value == Random
}

func (f FaultyModeValue) IsNotGossiped() bool {
	return f.Value == NotGossiped
}

func (f FaultyModeValue) IsSendWrongMsgType() bool {
	return f.Value == SendWrongMsgType
}

func (f FaultyModeValue) IsSendWrongMsgSeal() bool {
	return f.Value == SendWrongMsgSeal
}

func (f FaultyModeValue) IsSendWrongMsgSignature() bool {
	return f.Value == SendWrongMsgSeal
}

func (f FaultyModeValue) IsSendWrongMsgView() bool {
	return f.Value == SendWrongMsgView
}

func (f FaultyModeValue) IsSendWrongMsgDigest() bool {
	return f.Value == SendWrongMsgDigest
}

func (f FaultyModeValue) IsSendWrongMsgProposal() bool {
	return f.Value == SendWrongMsgProposal
}

func (f FaultyModeValue) IsAlwaysPropose() bool {
	return f.Value == AlwaysPropose
}

func (f FaultyModeValue) IsAlwaysRoundChange() bool {
	return f.Value == AlwaysRoundChange
}

func (f FaultyModeValue) IsBadBlock() bool {
	return f.Value == BadBlock
}

func (f FaultyModeValue) Uint64() uint64 {
	return uint64(f.Value)
}

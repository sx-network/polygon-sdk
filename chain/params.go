package chain

import (
	"math/big"
	"math/rand"
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
	// AlwaysRoundChange always sends as roundChange msg when msg type not roundChange
	AlwaysRoundChange = 3
	// NeverRoundChange always sends as commit msg when msg type roundChange
	NeverRoundChange = 4
	// SendWrongMsgSeal sends the message with a randomly-generated seal
	SendWrongMsgSeal = 5
	// SendWrongMsgSignature sends the message with a randomly-generated signature
	SendWrongMsgSignature = 6
	// SendWrongMsgView sends the message with a randomly-generated view
	SendWrongMsgView = 7
	// SendWrongMsgDigest sends the message with a randomly-generated digest
	SendWrongMsgDigest = 8
	// SendWrongMsgProposal sends the message with a randomly-generated proposal
	SendWrongMsgProposal = 9
	// AlwaysPropose always proposes a proposal to validators
	AlwaysPropose = 10
	// BadBlock always proposes a block with bad body
	BadBlock = 11
	// ScrambleState always return a state that we are not
	ScrambleState = 12
)

func (f FaultyModeValue) random() bool {
	return f.Value == Random && rand.Intn(2) == 1
}

func (f FaultyModeValue) IsNotGossiped() bool {
	return f.Value == NotGossiped || f.random()
}

func (f FaultyModeValue) IsAlwaysRoundChangeMsgType() bool {
	return f.Value == AlwaysRoundChange || f.random()
}

func (f FaultyModeValue) IsNeverRoundChangeMsgType() bool {
	return f.Value == NeverRoundChange || f.random()
}

func (f FaultyModeValue) IsSendWrongMsgSeal() bool {
	return f.Value == SendWrongMsgSeal || f.random()
}

func (f FaultyModeValue) IsSendWrongMsgSignature() bool {
	return f.Value == SendWrongMsgSignature || f.random()
}

func (f FaultyModeValue) IsSendWrongMsgView() bool {
	return f.Value == SendWrongMsgView || f.random()
}

func (f FaultyModeValue) IsSendWrongMsgDigest() bool {
	return f.Value == SendWrongMsgDigest || f.random()
}

func (f FaultyModeValue) IsSendWrongMsgProposal() bool {
	return f.Value == SendWrongMsgProposal || f.random()
}

func (f FaultyModeValue) IsAlwaysPropose() bool {
	return f.Value == AlwaysPropose || f.random()
}

func (f FaultyModeValue) IsBadBlock() bool {
	return f.Value == BadBlock || f.random()
}

func (f FaultyModeValue) IsScrambleState() bool {
	//TODO: enure this doesn't throw panic..
	//return f.Value == ScrambleState || f.random()
	return false 
}

func (f FaultyModeValue) Uint64() uint64 {
	return uint64(f.Value)
}

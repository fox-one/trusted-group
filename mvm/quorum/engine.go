package quorum

import (
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	ClockTick = 3 * time.Second
)

type Configuration struct {
	Store     string `toml:"store"`
	RPC       string `toml:"rpc"`
	Publisher bool   `toml:"publisher"`
}

type Engine struct {
	db        *badger.DB
	rpc       *RPC
	publisher bool
}

func Boot(conf *Configuration) (*Engine, error) {
	return nil, nil
}

func (e *Engine) VerifyAddress(address string) error {
	err := ethereum.VerifyAddress(address)
	if err != nil {
		return err
	}
	// format
	// ABI
	// TODO
	return nil
}

func (e *Engine) SetupNotifier(address string) error {
	// new private key based on address and a private key
	// use this private key to submit all events for this address
	// just add this command to storage
	// then another loop do all the works
	// TODO
	return nil
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	panic(0)
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	// local store write group events
	panic(0)
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	// rpc.eth_getLogs(fromBlock: offset, toBlock: offset+10, address: address, topics: groupTransfer)
	// TODO
	return nil, nil
}

func (e *Engine) loopSendGroupEvents(address string) {
	// for loop all group events
	// ensure the events are in RPC
	// batch events per transaction
	// there should be only one node engines send transactions
	// check events available before sending the transaction
	for e.publisher {
		time.Sleep(ClockTick)
	}
}

func (e *Engine) loopSetupNotifiers() {
	for {
	}
}

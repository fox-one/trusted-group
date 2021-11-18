package quorum

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

const (
	ClockTick = 3 * time.Second
	// event MixinTransaction(bytes);
	EventTopic = "0xdb53e751d28ed0d6e3682814bf8d23f7dd7b29c94f74a56fbb7f88e9dca9f39b"
	// function mixin(bytes calldata raw) public returns (bool)
	EventMethod = "0x5cae8005"

	ContractAgeLimit = 1
	GasLimit         = 100000000
	GasPrice         = 10000
)

type Configuration struct {
	Store      string `toml:"store"`
	RPC        string `toml:"rpc"`
	PrivateKey string `toml:"key"`
}

type Engine struct {
	db  *badger.DB
	rpc *RPC
	key string
}

func Boot(conf *Configuration) (*Engine, error) {
	db := openBadger(conf.Store)
	rpc, err := NewRPC(conf.RPC)
	if err != nil {
		return nil, err
	}
	e := &Engine{db: db, rpc: rpc}
	if conf.PrivateKey != "" {
		priv, err := crypto.HexToECDSA(conf.PrivateKey)
		if err != nil {
			panic(err)
		}
		e.key = hex.EncodeToString(crypto.FromECDSA(priv))
	}
	go e.loopHandleContracts()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) VerifyAddress(address string, hash []byte) error {
	err := ethereum.VerifyAddress(address)
	if err != nil {
		return err
	}
	height, err := e.rpc.GetBlockHeight()
	if err != nil {
		panic(err)
	}
	birth, err := e.rpc.GetContractBirthBlock(address, string(hash))
	if err != nil {
		return err
	}
	if height < birth+ContractAgeLimit {
		return fmt.Errorf("too young %d %d", birth, height)
	}
	// TODO ABI
	e.storeWriteContractLogsOffset(address, birth)
	return nil
}

func (e *Engine) SetupNotifier(address string) error {
	seed := e.Hash([]byte(e.key + address))
	key, err := crypto.ToECDSA(seed)
	if err != nil {
		panic(err)
	}
	notifier := hex.EncodeToString(crypto.FromECDSA(key))
	old := e.storeReadContractNotifier(address)
	if old == notifier {
		return nil
	} else if old != "" {
		panic(old)
	}
	return e.storeWriteContractNotifier(address, notifier)
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	// TODO should do it
	return common.Zero, nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	return e.storeListContractEvents(address, offset, limit)
}

func (e *Engine) IsPublisher() bool {
	return e.key != ""
}

func (e *Engine) loopGetLogs(address string) {
	nonce := e.storeReadLastContractEventNonce(address) + 1

	for {
		offset := e.storeReadContractLogsOffset(address)
		logs, err := e.rpc.GetLogs(address, EventTopic, offset, offset+10)
		if err != nil {
			panic(err)
		}
		var evts []*encoding.Event
		for _, b := range logs {
			evt, err := encoding.DecodeEvent(b)
			if err != nil {
				panic(err)
			}
			evts = append(evts, evt)
		}
		sort.Slice(evts, func(i, j int) bool { return evts[i].Nonce < evts[j].Nonce })
		for _, evt := range evts {
			if evt.Nonce < nonce {
				continue
			}
			if evt.Nonce > nonce {
				break
			}
			e.storeWriteContractEvent(address, evt)
			nonce = nonce + 1
		}
		e.storeWriteContractLogsOffset(address, offset+10)
		if len(logs) == 0 {
			time.Sleep(ClockTick * 5)
		}
	}
}

func (e *Engine) loopSendGroupEvents(address string) {
	notifier := e.storeReadContractNotifier(address)

	for e.IsPublisher() {
		balance, err := e.rpc.GetAddressBalance(pub(notifier))
		if err != nil {
			panic(err)
		}
		if balance.Cmp(decimal.NewFromInt(1)) < 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		nonce, err := e.rpc.GetAddressNonce(pub(notifier))
		if err != nil {
			panic(err)
		}
		evts, err := e.storeListGroupEvents(address, nonce, 100)
		if err != nil {
			panic(err)
		}
		for _, evt := range evts {
			id, raw := e.signGroupEventTransaction(address, evt, notifier)
			res, err := e.rpc.SendRawTransaction(raw)
			logger.Verbosef("SendRawTransaction(%s, %s) => %s, %v", id, raw, res, err)
		}
		if len(evts) == 0 {
			time.Sleep(ClockTick)
		}
	}
}

func (e *Engine) loopHandleContracts() {
	contracts := make(map[string]bool)
	for {
		all, err := e.storeListContractNotifiers()
		if err != nil {
			panic(err)
		}
		for _, c := range all {
			if contracts[c] {
				continue
			}
			contracts[c] = true
			e.loopGetLogs(c)
			e.loopSendGroupEvents(c)
		}

		nonce, err := e.rpc.GetAddressNonce(pub(e.key))
		if err != nil {
			panic(err)
		}
		for _, c := range all {
			notifier := e.storeReadContractNotifier(c)
			balance, err := e.rpc.GetAddressBalance(pub(notifier))
			if err != nil {
				panic(err)
			}
			if balance.Cmp(decimal.NewFromInt(10)) > 0 {
				continue
			}
			id, raw := e.signContractNotifierDepositTransaction(notifier, e.key, decimal.NewFromInt(100), nonce+1)
			res, err := e.rpc.SendRawTransaction(raw)
			logger.Verbosef("SendRawTransaction(%s, %s) => %s, %v", id, raw, res, err)
			nonce = nonce + 1
		}
	}
}

func pub(priv string) string {
	key, _ := crypto.HexToECDSA(priv)
	return crypto.PubkeyToAddress(key.PublicKey).Hex()
}

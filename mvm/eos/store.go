package eos

import (
	"encoding/binary"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixEosContractNotifier   = "EOS:CONTRACT:NOTIFIER:"
	prefixEosContractLogOffset  = "EOS:CONTRACT:LOG:OFFSET:"
	prefixEosContractEventQueue = "EOS:CONTRACT:EVENT:QUEUE:"
	prefixEosGroupEventQueue    = "EOS:GROUP:EVENT:QUEUE:"
	prefixTxRequestNonce        = "EOS:TXREQUEST:OFFSET:"
	prefixCurrentBlockNum       = "EOS:CURRENTBLOCKNUM:OFFSET:"
)

func (e *Engine) storeWriteContractNotifier(address, notifier string) error {
	key := []byte(prefixEosContractNotifier + address)
	return e.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			panic(address)
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		return txn.Set(key, []byte(notifier))
	})
}

func (e *Engine) storeReadContractNotifier(address string) string {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixEosContractNotifier + address)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return ""
	} else if err != nil {
		panic(err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	return string(val)
}

func (e *Engine) storeListContractAddresses() ([]string, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixEosContractNotifier)
	it := txn.NewIterator(opts)
	defer it.Close()

	var addresses []string
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := string(it.Item().Key())
		addr := key[len(prefixEosContractNotifier):]
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (e *Engine) storeReadCurrentBlockNum() uint64 {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixCurrentBlockNum)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint64(val)
}

func (e *Engine) storeWriteCurrentBlockNum(blockNum uint64) error {
	key := []byte(prefixCurrentBlockNum)
	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, uint64Bytes(blockNum))
	})
}

func (e *Engine) storeReadLastContractEventNonce(address string) uint64 {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(prefixEosContractEventQueue + address)
	opts.PrefetchValues = false
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(append(opts.Prefix, uint64Bytes(^uint64(0))...))
	if !it.Valid() {
		return 0
	}
	val, err := it.Item().ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	var evt encoding.Event
	err = encoding.JSONUnmarshal(val, &evt)
	if err != nil {
		panic(err)
	}
	return evt.Nonce
}

func (e *Engine) storeWriteContractEvent(address string, evt *encoding.Event) error {
	key := []byte(prefixEosContractEventQueue + address)
	key = append(key, uint64Bytes(evt.Nonce)...)
	val := encoding.JSONMarshalPanic(evt)
	return e.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			return nil
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		return txn.Set(key, val)
	})
}

func (e *Engine) storeListContractEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixEosContractEventQueue + address)
	it := txn.NewIterator(opts)
	defer it.Close()

	var events []*encoding.Event
	it.Seek(append(opts.Prefix, uint64Bytes(offset)...))
	for ; it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = encoding.JSONUnmarshal(val, &evt)
		if err != nil {
			panic(err)
		}
		events = append(events, &evt)
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

func (e *Engine) storeWriteGroupEvents(address string, events []*encoding.Event) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for _, evt := range events {
			key := []byte(prefixEosGroupEventQueue + address)
			key = append(key, uint64Bytes(evt.Nonce)...)
			val := encoding.JSONMarshalPanic(evt)
			_, err := txn.Get(key)
			if err == nil {
				continue
			} else if err != badger.ErrKeyNotFound {
				return err
			}
			err = txn.Set(key, val)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Engine) storeListGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixEosGroupEventQueue + address)
	it := txn.NewIterator(opts)
	defer it.Close()

	var events []*encoding.Event
	it.Seek(append(opts.Prefix, uint64Bytes(offset)...))
	for ; it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = encoding.JSONUnmarshal(val, &evt)
		if err != nil {
			panic(err)
		}
		events = append(events, &evt)
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

func uint64Bytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

func openBadger(dir string) *badger.DB {
	opts := badger.DefaultOptions(dir)
	db, err := badger.Open(opts)
	if err != nil {
		panic(err)
	}
	return db
}

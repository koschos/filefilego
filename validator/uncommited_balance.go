// Package validator implements sealing and broadcasting blocks to the network.
//
// The block sealing process is to go through all the transactions in the mempool
// and order them by nounce and later on construct an uncommitted balance which will
// be used to check if a transaction has enough balance and allowed to change the
// state of the blockchain.
package validator

import (
	"math/big"
	"sync"
)

type balanceItem struct {
	dbBalance *big.Int
	dbNounce  uint64
}

// UncommitedBalance represents an uncommitted balance of a user.
type UncommitedBalance struct {
	addresses map[string]balanceItem
	mu        sync.RWMutex
}

// InitializeBalanceAndNounceFor sets the balance and nounce for an address.
func (b *UncommitedBalance) InitializeBalanceAndNounceFor(address string, balance *big.Int, nounce uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, ok := b.addresses[address]
	if ok {
		return
	}
	data.dbBalance = balance
	data.dbNounce = nounce
	b.addresses[address] = data
}

// IsInitialized checks if balance is initialized for an address.
func (b *UncommitedBalance) IsInitialized(address string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.addresses[address]
	return ok
}

// Subtract from balance.
func (b *UncommitedBalance) Subtract(address string, amount *big.Int, nounce uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, ok := b.addresses[address]
	if !ok {
		return false
	}

	if data.dbBalance.Cmp(amount) == -1 {
		return false
	}

	if nounce != data.dbNounce+1 {
		return false
	}

	data.dbBalance = data.dbBalance.Sub(data.dbBalance, amount)
	data.dbNounce = nounce

	b.addresses[address] = data
	return true
}

// NewUncommitedBalance creates a new uncommitted balance.
func NewUncommitedBalance() UncommitedBalance {
	return UncommitedBalance{
		addresses: make(map[string]balanceItem),
	}
}

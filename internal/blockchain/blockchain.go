package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	sync "sync"

	"github.com/filefilego/filefilego/internal/block"
	"github.com/filefilego/filefilego/internal/common/hexutil"
	"github.com/filefilego/filefilego/internal/database"
	"github.com/filefilego/filefilego/internal/transaction"
)

// Blockchain represents a blockchain.
// type Blockchain interface {
// 	AddHeight(h uint64)
// 	GetHeight() uint64

// 	// we need boltdb bucket for hash->tx
// 	GetTransactionByHash(hash string) (tx transaction.Transaction, block block.Block, index uint64, err error)
// 	// we need a boltdb bucket for addr->txhash
// 	GetTransactionsByAddress(address string) (txs []transaction.Transaction, err error)
// 	// we need a bucket for height->block
// 	GetBlockByHeight(number uint64) (block block.Block, err error)
// 	// this will be fixed if we implement the above
// 	GetBlocksByRange(from uint64, to uint64) ([]block.Block, error)
// 	// we need a bucket for hash->blockchain
// 	GetBlockByHash(hash string) (block block.Block, err error)

// 	AddBalanceTo(address string, amount *big.Int) error
// 	SubBalanceOf(address string, amount *big.Int, nounce string) error

// 	MutateChannel(t transaction.Transaction, vbalances map[string]*big.Int, isMiningMode bool) error
// 	MutateAddressStateFromTransaction(transaction transaction.Transaction, isCoinbase bool) (err error)
// 	HasThisBalance(address string, amount *big.Int) (bool, *big.Int, *big.Int, error)

// 	// already hanled by transaction package
// 	SignTransaction(transaction transaction.Transaction, keystroe string) (transaction.Transaction, error)
// 	IsValidTransaction(transaction transaction.Transaction) (bool, error)

// 	GetNounceFromMemPool(address string) (string, error)

// 	AddBlockPool(block block.Block) (bool, error)
// 	removeBlockPool(block block.Block) error
// 	ClearBlockPool(lock bool)

// 	AddMemPool(transaction transaction.Transaction) error
// 	RemoveMemPool(transaction transaction.Transaction) error

// 	// below one calls the other
// 	PersistMemPoolToDB() error
// 	SerializeMemPool() ([]byte, error)

// 	LoadToMemPoolFromDB()

// 	MineBlock(transactions []transaction.Transaction) (block.Block, error)
// 	// GetAddressData(address string) (ads AddressState, merr AddressDataResult)
// 	// TraverseChanNodes(hash []byte, fn transformNode)
// 	TraverseChain(fn transform)
// 	PreparePoolBlocksForMining() ([]transaction.Transaction, map[string]*big.Int)
// 	CalculateReward() string
// 	MineScheduler()
// }

// type transform func(block.Block)
// type transformNode func(ChanNode)
// type AddressDataResult int

const addressPrefix = "address"

const blockPrefix = "blocks"

const lastBlockPrefix = "last_block"

// Interface wraps the functionality of a blockchain.
type Interface interface {
	GetBlocksFromPool() []block.Block
	PutBlockPool(block block.Block) error
	DeleteFromBlockPool(block block.Block) error
	PutMemPool(tx transaction.Transaction) error
	DeleteFromMemPool(tx transaction.Transaction) error
	GetTransactionsFromPool() []transaction.Transaction
	SaveBlockInDB(blck block.Block) error
	GetBlockByHash(blockHash []byte) (block.Block, error)
	GetAddressState(address []byte) (AddressState, error)
	UpdateAddressState(address []byte, state AddressState) error
	CloseDB() error
	IncrementHeightBy(h uint64)
	GetHeight() uint64
}

// Blockchain represents a blockchain structure.
type Blockchain struct {
	db        database.Database
	blockPool map[string]block.Block
	bmu       sync.RWMutex

	memPool map[string]transaction.Transaction
	tmu     sync.RWMutex

	height uint64
	hmu    sync.RWMutex

	lastBlockHash []byte
}

// New creates a new blockchain instance.
func New(db database.Database) (*Blockchain, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &Blockchain{
		db:            db,
		blockPool:     make(map[string]block.Block),
		memPool:       make(map[string]transaction.Transaction),
		lastBlockHash: make([]byte, 0),
	}, nil
}

// InitOrLoad increments the blockchain height by the given number.
func (b *Blockchain) InitOrLoad() error {
	lastBlockHash, err := b.db.Get([]byte(lastBlockPrefix))
	if err != nil && len(lastBlockHash) == 0 {
		// init blockchain
		return nil
	}

	// load blockchain
	// logic

	return nil
}

// IncrementHeightBy increments the blockchain height by the given number.
func (b *Blockchain) IncrementHeightBy(h uint64) {
	b.hmu.Lock()
	defer b.hmu.Unlock()

	b.height += h
}

// GetHeight gets the height of the blockchain.
func (b *Blockchain) GetHeight() uint64 {
	b.hmu.RLock()
	defer b.hmu.RUnlock()

	return b.height
}

// GetBlocksFromPool get all the block from blockpool.
func (b *Blockchain) GetBlocksFromPool() []block.Block {
	b.bmu.RLock()
	defer b.bmu.RUnlock()

	blocks := make([]block.Block, 0, len(b.blockPool))
	for _, blc := range b.blockPool {
		blocks = append(blocks, blc)
	}

	return blocks
}

// PutBlockPool adds a block to blockPool.
func (b *Blockchain) PutBlockPool(block block.Block) error {
	b.bmu.Lock()
	defer b.bmu.Unlock()

	blockHash := hexutil.Encode(block.Hash)
	b.blockPool[blockHash] = block
	return nil
}

// DeleteFromBlockPool deletes a block from mempool.
func (b *Blockchain) DeleteFromBlockPool(block block.Block) error {
	b.bmu.Lock()
	defer b.bmu.Unlock()

	blockHash := hexutil.Encode(block.Hash)
	delete(b.blockPool, blockHash)
	return nil
}

// addBalanceTo adds balance to address.
func (b *Blockchain) addBalanceTo(address []byte, amount *big.Int) error {
	zeroBig := big.NewInt(0)
	if amount.Cmp(zeroBig) == -1 {
		return errors.New("amount is negative")
	}
	state, err := b.GetAddressState(address)
	// address has no balance
	if err != nil {
		err := state.SetBalance(big.NewInt(0))
		if err != nil {
			return fmt.Errorf("failed to set balance: %w", err)
		}
		err = state.SetNounce(0)
		if err != nil {
			return fmt.Errorf("failed to set nounce: %w", err)
		}
	}

	balance, err := state.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	err = state.SetBalance(balance.Add(balance, amount))
	if err != nil {
		return fmt.Errorf("failed to set balance: %w", err)
	}

	err = b.UpdateAddressState(address, state)
	if err != nil {
		return fmt.Errorf("failed to update balance state: %w", err)
	}

	return nil
}

// subBalanceFrom subtracts balance from address.
func (b *Blockchain) subBalanceFrom(address []byte, amount *big.Int, nounce uint64) error {
	zeroBig := big.NewInt(0)
	if amount.Cmp(zeroBig) == -1 {
		return errors.New("amount is negative")
	}
	state, err := b.GetAddressState(address)
	// address has no balance
	if err != nil {
		return fmt.Errorf("address has no balance: %w", err)
	}

	balance, err := state.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(amount) < 0 {
		return errors.New("failed to subtract: amount is greater than balance")
	}

	err = state.SetBalance(balance.Sub(balance, amount))
	if err != nil {
		return fmt.Errorf("failed to set balance: %w", err)
	}

	err = state.SetNounce(nounce)
	if err != nil {
		return fmt.Errorf("failed to set nounce: %w", err)
	}

	err = b.UpdateAddressState(address, state)
	if err != nil {
		return fmt.Errorf("failed to update balance state: %w", err)
	}

	return nil
}

// PerformAddressStateUpdate performs state update.
// This function should be able to rollback to previous state in case of failure.
// APPLYING OPERATIONS ON BIG INTS MODIFIES THE UNDERLYING DATA.
func (b *Blockchain) PerformAddressStateUpdate(transaction transaction.Transaction, verifierAddr []byte, isCoinbase bool) error {
	ok, err := transaction.Validate()
	if err != nil || !ok {
		return fmt.Errorf("failed to validate transaction: %w", err)
	}
	txFees, err := hexutil.DecodeBig(transaction.TransactionFees)
	if err != nil {
		return fmt.Errorf("failed to decode transaction fees: %w", err)
	}

	// if not coinbase tx, then subtract the amount from the account
	if !isCoinbase {
		txValue, err := hexutil.DecodeBig(transaction.Value)
		if err != nil {
			return fmt.Errorf("failed to decode transaction value: %w", err)
		}
		totalFees := txValue.Add(txValue, txFees)
		addrBytes, err := hexutil.Decode(transaction.From)
		if err != nil {
			return fmt.Errorf("failed to decode from address: %w", err)
		}

		err = b.subBalanceFrom(addrBytes, totalFees, hexutil.DecodeBigFromBytesToUint64(transaction.Nounce))
		if err != nil {
			return fmt.Errorf("failed to subtract total value from address: %w", err)
		}
	}

	toAddrBytes, err := hexutil.Decode(transaction.To)
	if err != nil {
		return fmt.Errorf("failed to decode to address: %w", err)
	}

	txValue, err := hexutil.DecodeBig(transaction.Value)
	if err != nil {
		return fmt.Errorf("failed to decode transaction value: %w", err)
	}

	err = b.addBalanceTo(toAddrBytes, txValue)
	if err != nil {
		return fmt.Errorf("failed to add amount to balance: %w", err)
	}

	err = b.addBalanceTo(verifierAddr, txFees)
	if err != nil {
		return fmt.Errorf("failed to add amount to verifier's balance: %w", err)
	}

	err = b.performStateUpdateFromDataPayload(transaction.Data)
	if err != nil {
		return fmt.Errorf("failed perform state update from transaction data: %w", err)
	}

	return nil
}

// performStateUpdateFromDataPayload performs updates from the transaction data.
func (b *Blockchain) performStateUpdateFromDataPayload(dataPayload []byte) error {
	return nil
}

// GetNounceFromMemPool get the nounce from mempool for an address.
func (b *Blockchain) GetNounceFromMemPool(address []byte) uint64 {
	b.tmu.RLock()
	defer b.tmu.RUnlock()

	tmp := uint64(0)
	for _, v := range b.memPool {
		if v.From == hexutil.Encode(address) {
			nounce := hexutil.DecodeBigFromBytesToUint64(v.Nounce)
			if nounce > tmp {
				tmp = nounce
			}
		}
	}

	return tmp
}

// PutMemPool adds a transaction to mempool.
func (b *Blockchain) PutMemPool(tx transaction.Transaction) error {
	b.tmu.Lock()
	defer b.tmu.Unlock()

	// TODO: handle the case when there is a tx with lower nounce than the one in db

	for idx, transaction := range b.memPool {
		// transaction is already in mempool with this nounce
		// pick the one with higher fee
		if bytes.Equal(transaction.Nounce, tx.Nounce) && transaction.From == tx.From {
			txFees, err := hexutil.DecodeBig(tx.TransactionFees)
			if err != nil {
				return fmt.Errorf("failed to decode transaction fees: %w", err)
			}
			txFeesInMempool, err := hexutil.DecodeBig(transaction.TransactionFees)
			if err != nil {
				return fmt.Errorf("failed to decode transaction fees from mempool: %w", err)
			}

			if txFees.Cmp(txFeesInMempool) == 1 {
				b.memPool[idx] = tx
				return nil
			}
		}
	}

	txHash := hexutil.Encode(tx.Hash)
	b.memPool[txHash] = tx
	return nil
}

// DeleteFromMemPool deletes a transaction from mempool.
func (b *Blockchain) DeleteFromMemPool(tx transaction.Transaction) error {
	b.tmu.Lock()
	defer b.tmu.Unlock()

	txHash := hexutil.Encode(tx.Hash)
	delete(b.memPool, txHash)
	return nil
}

// GetTransactionsFromPool get all the transactions from mempool.
func (b *Blockchain) GetTransactionsFromPool() []transaction.Transaction {
	b.tmu.RLock()
	defer b.tmu.RUnlock()

	txs := make([]transaction.Transaction, 0, len(b.memPool))
	for _, tx := range b.memPool {
		txs = append(txs, tx)
	}

	return txs
}

// SaveBlockInDB saves a block into the database.
func (b *Blockchain) SaveBlockInDB(blck block.Block) error {
	if len(blck.Hash) == 0 {
		return errors.New("blockhash is empty")
	}
	protoblock := block.ToProtoBlock(blck)
	data, err := block.MarshalProtoBlock(protoblock)
	if err != nil {
		return fmt.Errorf("failed to marshal protoblock: %w", err)
	}
	err = b.db.Put(append([]byte(blockPrefix), blck.Hash...), data)
	if err != nil {
		return fmt.Errorf("failed to save data into db: %w", err)
	}
	return nil
}

// GetBlockByHash gets a block by its hash.
func (b *Blockchain) GetBlockByHash(blockHash []byte) (block.Block, error) {
	if len(blockHash) == 0 {
		return block.Block{}, errors.New("blockhash is empty")
	}

	blockData, err := b.db.Get(append([]byte(blockPrefix), blockHash...))
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to get block from database: %w", err)
	}

	protoBlock, err := block.UnmarshalProtoBlock(blockData)
	if err != nil {
		return block.Block{}, fmt.Errorf("failed to get unmarshal protoblock: %w", err)
	}

	return block.ProtoBlockToBlock(protoBlock), nil
}

// GetAddressState returns the state of the address from the db.
func (b *Blockchain) GetAddressState(address []byte) (AddressState, error) {
	data, err := b.db.Get(append([]byte(addressPrefix), address...))
	if err != nil {
		return AddressState{}, fmt.Errorf("failed to get address state: %w", err)
	}
	protoAddrState, err := UnmarshalAddressStateProto(data)
	if err != nil {
		return AddressState{}, fmt.Errorf("failed to unmarshal address state: %w", err)
	}
	return AddressStateProtoToAddressState(protoAddrState), nil
}

// UpdateAddressState updates the state of the address in the db.
func (b *Blockchain) UpdateAddressState(address []byte, state AddressState) error {
	if len(address) == 0 {
		return errors.New("address is empty")
	}

	protoAddrState := ToAddressStateProto(state)
	data, err := MarshalAddressStateProto(protoAddrState)
	if err != nil {
		return fmt.Errorf("failed to marshal address state: %w", err)
	}

	err = b.db.Put(append([]byte(addressPrefix), address...), data)
	if err != nil {
		return fmt.Errorf("failed to put to database: %w", err)
	}

	return nil
}

// CloseDB closes the db.
func (b *Blockchain) CloseDB() error {
	return b.db.Close()
}

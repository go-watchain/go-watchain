// Copyright 2015 The go-ethereum Authors
// This file is part of the go-watereum library.
//
// The go-watereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-watereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-watereum library. If not, see <http://www.gnu.org/licenses/>.

package wat

import (
	"context"
	"math/big"

	"github.com/watchain/go-watchain/accounts"
	"github.com/watchain/go-watchain/common"
	"github.com/watchain/go-watchain/common/math"
	"github.com/watchain/go-watchain/core"
	"github.com/watchain/go-watchain/core/bloombits"
	"github.com/watchain/go-watchain/core/state"
	"github.com/watchain/go-watchain/core/types"
	"github.com/watchain/go-watchain/core/vm"
	"github.com/watchain/go-watchain/wat/downloader"
	"github.com/watchain/go-watchain/wat/gasprice"
	"github.com/watchain/go-watchain/watdb"
	"github.com/watchain/go-watchain/event"
	"github.com/watchain/go-watchain/params"
	"github.com/watchain/go-watchain/rpc"
)

// watApiBackend implements ethapi.Backend for full nodes
type watApiBackend struct {
	wat *watchain
	gpo *gasprice.Oracle
}

func (b *watApiBackend) ChainConfig() *params.ChainConfig {
	return b.wat.chainConfig
}

func (b *watApiBackend) CurrentBlock() *types.Block {
	return b.wat.blockchain.CurrentBlock()
}

func (b *watApiBackend) SetHead(number uint64) {
	b.wat.protocolManager.downloader.Cancel()
	b.wat.blockchain.SetHead(number)
}

func (b *watApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.wat.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.wat.blockchain.CurrentBlock().Header(), nil
	}
	return b.wat.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *watApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.wat.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.wat.blockchain.CurrentBlock(), nil
	}
	return b.wat.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *watApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.wat.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.wat.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *watApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.wat.blockchain.GetBlockByHash(blockHash), nil
}

func (b *watApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.wat.chainDb, blockHash, core.GetBlockNumber(b.wat.chainDb, blockHash)), nil
}

func (b *watApiBackend) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := core.GetBlockReceipts(b.wat.chainDb, blockHash, core.GetBlockNumber(b.wat.chainDb, blockHash))
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *watApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.wat.blockchain.GetTdByHash(blockHash)
}

func (b *watApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.wat.BlockChain(), nil)
	return vm.NewEVM(context, state, b.wat.chainConfig, vmCfg), vmError, nil
}

func (b *watApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.wat.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *watApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.wat.BlockChain().SubscribeChainEvent(ch)
}

func (b *watApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.wat.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *watApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.wat.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *watApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.wat.BlockChain().SubscribeLogsEvent(ch)
}

func (b *watApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.wat.txPool.AddLocal(signedTx)
}

func (b *watApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.wat.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *watApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.wat.txPool.Get(hash)
}

func (b *watApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.wat.txPool.State().GetNonce(addr), nil
}

func (b *watApiBackend) Stats() (pending int, queued int) {
	return b.wat.txPool.Stats()
}

func (b *watApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.wat.TxPool().Content()
}

func (b *watApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.wat.TxPool().SubscribeTxPreEvent(ch)
}

func (b *watApiBackend) Downloader() *downloader.Downloader {
	return b.wat.Downloader()
}

func (b *watApiBackend) ProtocolVersion() int {
	return b.wat.watVersion()
}

func (b *watApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *watApiBackend) ChainDb() watdb.Database {
	return b.wat.ChainDb()
}

func (b *watApiBackend) EventMux() *event.TypeMux {
	return b.wat.EventMux()
}

func (b *watApiBackend) AccountManager() *accounts.Manager {
	return b.wat.AccountManager()
}

func (b *watApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.wat.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *watApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.wat.bloomRequests)
	}
}

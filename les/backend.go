// Copyright 2016 The go-ethereum Authors
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

// Package les implements the Light watchain Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/watchain/go-watchain/accounts"
	"github.com/watchain/go-watchain/common"
	"github.com/watchain/go-watchain/common/hexutil"
	"github.com/watchain/go-watchain/consensus"
	"github.com/watchain/go-watchain/core"
	"github.com/watchain/go-watchain/core/bloombits"
	"github.com/watchain/go-watchain/core/types"
	"github.com/watchain/go-watchain/wat"
	"github.com/watchain/go-watchain/wat/downloader"
	"github.com/watchain/go-watchain/wat/filters"
	"github.com/watchain/go-watchain/wat/gasprice"
	"github.com/watchain/go-watchain/watdb"
	"github.com/watchain/go-watchain/event"
	"github.com/watchain/go-watchain/internal/ethapi"
	"github.com/watchain/go-watchain/light"
	"github.com/watchain/go-watchain/log"
	"github.com/watchain/go-watchain/node"
	"github.com/watchain/go-watchain/p2p"
	"github.com/watchain/go-watchain/p2p/discv5"
	"github.com/watchain/go-watchain/params"
	rpc "github.com/watchain/go-watchain/rpc"
)

type Lightwatchain struct {
	config *wat.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb watdb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *wat.Config) (*Lightwatchain, error) {
	chainDb, err := wat.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lwat := &Lightwatchain{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           wat.CreateConsensusEngine(ctx, &config.watash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     wat.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lwat.relay = NewLesTxRelay(peers, lwat.reqDist)
	lwat.serverPool = newServerPool(chainDb, quitSync, &lwat.wg)
	lwat.retriever = newRetrieveManager(peers, lwat.reqDist, lwat.serverPool)
	lwat.odr = NewLesOdr(chainDb, lwat.chtIndexer, lwat.bloomTrieIndexer, lwat.bloomIndexer, lwat.retriever)
	if lwat.blockchain, err = light.NewLightChain(lwat.odr, lwat.chainConfig, lwat.engine); err != nil {
		return nil, err
	}
	lwat.bloomIndexer.Start(lwat.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lwat.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lwat.txPool = light.NewTxPool(lwat.chainConfig, lwat.blockchain, lwat.relay)
	if lwat.protocolManager, err = NewProtocolManager(lwat.chainConfig, true, ClientProtocolVersions, config.NetworkId, lwat.eventMux, lwat.engine, lwat.peers, lwat.blockchain, nil, chainDb, lwat.odr, lwat.relay, quitSync, &lwat.wg); err != nil {
		return nil, err
	}
	lwat.ApiBackend = &LesApiBackend{lwat, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lwat.ApiBackend.gpo = gasprice.NewOracle(lwat.ApiBackend, gpoParams)
	return lwat, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// waterbase is the address that mining rewards will be send to
func (s *LightDummyAPI) waterbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for waterbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the watereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Lightwatchain) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "wat",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "wat",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "wat",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Lightwatchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Lightwatchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *Lightwatchain) TxPool() *light.TxPool              { return s.txPool }
func (s *Lightwatchain) Engine() consensus.Engine           { return s.engine }
func (s *Lightwatchain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Lightwatchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Lightwatchain) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Lightwatchain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// watchain protocol implementation.
func (s *Lightwatchain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// watchain protocol.
func (s *Lightwatchain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}

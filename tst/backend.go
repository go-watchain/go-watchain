// Copyright 2014 The go-ethereum Authors
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

// Package wat implements the watchain protocol.
package wat

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/watchain/go-watchain/accounts"
	"github.com/watchain/go-watchain/common"
	"github.com/watchain/go-watchain/common/hexutil"
	"github.com/watchain/go-watchain/consensus"
	"github.com/watchain/go-watchain/consensus/clique"
	"github.com/watchain/go-watchain/consensus/ethash"
	"github.com/watchain/go-watchain/core"
	"github.com/watchain/go-watchain/core/bloombits"
	"github.com/watchain/go-watchain/core/types"
	"github.com/watchain/go-watchain/core/vm"
	"github.com/watchain/go-watchain/wat/downloader"
	"github.com/watchain/go-watchain/wat/filters"
	"github.com/watchain/go-watchain/wat/gasprice"
	"github.com/watchain/go-watchain/watdb"
	"github.com/watchain/go-watchain/event"
	"github.com/watchain/go-watchain/internal/ethapi"
	"github.com/watchain/go-watchain/log"
	"github.com/watchain/go-watchain/miner"
	"github.com/watchain/go-watchain/node"
	"github.com/watchain/go-watchain/p2p"
	"github.com/watchain/go-watchain/params"
	"github.com/watchain/go-watchain/rlp"
	"github.com/watchain/go-watchain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// watchain implements the watchain full node service.
type watchain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the watereum
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb watdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *watApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	waterbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and waterbase)
}

func (s *watchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new watchain object (including the
// initialisation of the common watchain object)
func New(ctx *node.ServiceContext, config *Config) (*watchain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run wat.watchain in light sync mode, use les.Lightwatchain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	wat := &watchain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.watash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		waterbase:      config.waterbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising watchain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gwat upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	wat.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, wat.chainConfig, wat.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		wat.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	wat.bloomIndexer.Start(wat.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	wat.txPool = core.NewTxPool(config.TxPool, wat.chainConfig, wat.blockchain)

	if wat.protocolManager, err = NewProtocolManager(wat.chainConfig, config.SyncMode, config.NetworkId, wat.eventMux, wat.txPool, wat.engine, wat.blockchain, chainDb); err != nil {
		return nil, err
	}
	wat.miner = miner.New(wat, wat.chainConfig, wat.EventMux(), wat.engine)
	wat.miner.SetExtra(makeExtraData(config.ExtraData))

	wat.ApiBackend = &watApiBackend{wat, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	wat.ApiBackend.gpo = gasprice.NewOracle(wat.ApiBackend, gpoParams)

	return wat, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gwat",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (watdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*watdb.LDBDatabase); ok {
		db.Meter("wat/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an watchain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db watdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == ethash.ModeFake:
		log.Warn("watash used in fake mode")
		return ethash.NewFaker()
	case config.PowMode == ethash.ModeTest:
		log.Warn("watash used in test mode")
		return ethash.NewTester()
	case config.PowMode == ethash.ModeShared:
		log.Warn("watash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the watereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *watchain) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "wat",
			Version:   "1.0",
			Service:   NewPublicwatchainAPI(s),
			Public:    true,
		}, {
			Namespace: "wat",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "wat",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "wat",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *watchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *watchain) waterbase() (eb common.Address, err error) {
	s.lock.RLock()
	waterbase := s.waterbase
	s.lock.RUnlock()

	if waterbase != (common.Address{}) {
		return waterbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			waterbase := accounts[0].Address

			s.lock.Lock()
			s.waterbase = waterbase
			s.lock.Unlock()

			log.Info("waterbase automatically configured", "address", waterbase)
			return waterbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("waterbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *watchain) Setwaterbase(waterbase common.Address) {
	self.lock.Lock()
	self.waterbase = waterbase
	self.lock.Unlock()

	self.miner.Setwaterbase(waterbase)
}

func (s *watchain) StartMining(local bool) error {
	eb, err := s.waterbase()
	if err != nil {
		log.Error("Cannot start mining without waterbase", "err", err)
		return fmt.Errorf("waterbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("waterbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *watchain) StopMining()         { s.miner.Stop() }
func (s *watchain) IsMining() bool      { return s.miner.Mining() }
func (s *watchain) Miner() *miner.Miner { return s.miner }

func (s *watchain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *watchain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *watchain) TxPool() *core.TxPool               { return s.txPool }
func (s *watchain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *watchain) Engine() consensus.Engine           { return s.engine }
func (s *watchain) ChainDb() watdb.Database            { return s.chainDb }
func (s *watchain) IsListening() bool                  { return true } // Always listening
func (s *watchain) watVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *watchain) NetVersion() uint64                 { return s.networkId }
func (s *watchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *watchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// watchain protocol implementation.
func (s *watchain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// watchain protocol.
func (s *watchain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}

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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gwat

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/watchain/go-watchain/core"
	"github.com/watchain/go-watchain/wat"
	"github.com/watchain/go-watchain/wat/downloader"
	"github.com/watchain/go-watchain/watclient"
	"github.com/watchain/go-watchain/watstats"
	"github.com/watchain/go-watchain/les"
	"github.com/watchain/go-watchain/node"
	"github.com/watchain/go-watchain/p2p"
	"github.com/watchain/go-watchain/p2p/nat"
	"github.com/watchain/go-watchain/params"
	whisper "github.com/watchain/go-watchain/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Gwat
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-watereum to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Boowatrap nodes used to establish connectivity with the rest of the network.
	BoowatrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// watchainEnabled specifies whwater the node should run the watchain protocol.
	watchainEnabled bool

	// watchainNetworkID is the network identifier used by the watchain protocol to
	// decide if remote peers should be accepted or not.
	watchainNetworkID int64 // uint64 in truth, but Java can't handle that...

	// watchainGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	watchainGenesis string

	// watchainDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	watchainDatabaseCache int

	// watchainNewatats is a newatats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	watchainNewatats string

	// WhisperEnabled specifies whwater the node should run the Whisper protocol.
	WhisperEnabled bool
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BoowatrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	watchainEnabled:       true,
	watchainNetworkID:     1,
	watchainDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gwat watchain node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gwat node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BoowatrapNodes == nil || config.BoowatrapNodes.Size() == 0 {
		config.BoowatrapNodes = defaultNodeConfig.BoowatrapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BoowatrapNodesV5: config.BoowatrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.watchainGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.watchainGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.watchainGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.watchainNetworkID == 1 {
				config.watchainNetworkID = 3
			}
		}
	}
	// Register the watchain protocol if requested
	if config.watchainEnabled {
		watConf := wat.DefaultConfig
		watConf.Genesis = genesis
		watConf.SyncMode = downloader.LightSync
		watConf.NetworkId = uint64(config.watchainNetworkID)
		watConf.DatabaseCache = config.watchainDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &watConf)
		}); err != nil {
			return nil, fmt.Errorf("watereum init: %v", err)
		}
		// If newatats reporting is requested, do it
		if config.watchainNewatats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.Lightwatchain
				ctx.Service(&lesServ)

				return watstats.New(config.watchainNewatats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("newatats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetwatchainClient retrieves a client to access the watchain subsystem.
func (n *Node) GetwatchainClient() (client *watchainClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &watchainClient{watclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}

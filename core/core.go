package core

import (
	"encoding/base64"
	"errors"
	"fmt"

	context "github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"
	b58 "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-base58"
	ds "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	ma "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multiaddr"

	bserv "github.com/jbenet/go-ipfs/blockservice"
	config "github.com/jbenet/go-ipfs/config"
	diag "github.com/jbenet/go-ipfs/diagnostics"
	exchange "github.com/jbenet/go-ipfs/exchange"
	bitswap "github.com/jbenet/go-ipfs/exchange/bitswap"
	merkledag "github.com/jbenet/go-ipfs/merkledag"
	namesys "github.com/jbenet/go-ipfs/namesys"
	inet "github.com/jbenet/go-ipfs/net"
	mux "github.com/jbenet/go-ipfs/net/mux"
	netservice "github.com/jbenet/go-ipfs/net/service"
	path "github.com/jbenet/go-ipfs/path"
	peer "github.com/jbenet/go-ipfs/peer"
	pin "github.com/jbenet/go-ipfs/pin"
	routing "github.com/jbenet/go-ipfs/routing"
	dht "github.com/jbenet/go-ipfs/routing/dht"
	u "github.com/jbenet/go-ipfs/util"
)

var log = u.Logger("core")

// IpfsNode is IPFS Core module. It represents an IPFS instance.
type IpfsNode struct {

	// the node's configuration
	Config *config.Config

	// the local node's identity
	Identity peer.Peer

	// storage for other Peer instances
	Peerstore peer.Peerstore

	// the local datastore
	Datastore ds.ThreadSafeDatastore

	// the network message stream
	Network inet.Network

	// the routing system. recommend ipfs-dht
	Routing routing.IpfsRouting

	// the block exchange + strategy (bitswap)
	Exchange exchange.Interface

	// the block service, get/add blocks.
	Blocks *bserv.BlockService

	// the merkle dag service, get/add objects.
	DAG merkledag.DAGService

	// the path resolution system
	Resolver *path.Resolver

	// the name system, resolves paths to hashes
	Namesys namesys.NameSystem

	// the diagnostics service
	Diagnostics *diag.Diagnostics

	// the pinning manager
	Pinning pin.Pinner
}

// NewIpfsNode constructs a new IpfsNode based on the given config.
func NewIpfsNode(cfg *config.Config, online bool) (*IpfsNode, error) {
	// derive this from a higher context.
	// cancel if we need to fail early.
	ctx, cancel := context.WithCancel(context.TODO())
	success := false // flip to true after all sub-system inits succeed
	defer func() {
		if !success {
			cancel()
		}
	}()

	if cfg == nil {
		return nil, fmt.Errorf("configuration required")
	}

	d, err := makeDatastore(cfg.Datastore)
	if err != nil {
		return nil, err
	}

	peerstore := peer.NewPeerstore()
	local, err := initIdentity(cfg, peerstore, online)
	if err != nil {
		return nil, err
	}

	// FIXME(brian): This is a bit dangerous. If any of the vars declared in
	// this block are assigned inside of the "if online" block using the ":="
	// declaration syntax, the compiler permits re-declaration. This is rather
	// undesirable
	var (
		net inet.Network
		// TODO: refactor so we can use IpfsRouting interface instead of being DHT-specific
		route           *dht.IpfsDHT
		exchangeSession exchange.Interface
		diagnostics     *diag.Diagnostics
		network         inet.Network
	)

	if online {

		dhtService := netservice.NewService(ctx, nil)      // nil handler for now, need to patch it
		exchangeService := netservice.NewService(ctx, nil) // nil handler for now, need to patch it
		diagService := netservice.NewService(ctx, nil)

		net, err = inet.NewIpfsNetwork(ctx, local, peerstore, &mux.ProtocolMap{
			mux.ProtocolID_Routing:    dhtService,
			mux.ProtocolID_Exchange:   exchangeService,
			mux.ProtocolID_Diagnostic: diagService,
			// add protocol services here.
		})
		if err != nil {
			return nil, err
		}
		network = net

		diagnostics = diag.NewDiagnostics(local, net, diagService)
		diagService.SetHandler(diagnostics)

		route = dht.NewDHT(ctx, local, peerstore, net, dhtService, d)
		// TODO(brian): perform this inside NewDHT factory method
		dhtService.SetHandler(route) // wire the handler to the service.

		const alwaysSendToPeer = true // use YesManStrategy
		exchangeSession = bitswap.NetMessageSession(ctx, local, net, exchangeService, route, d, alwaysSendToPeer)

		// TODO(brian): pass a context to initConnections
		go initConnections(ctx, cfg, peerstore, route)
	}

	// TODO(brian): when offline instantiate the BlockService with a bitswap
	// session that simply doesn't return blocks
	bs, err := bserv.NewBlockService(d, exchangeSession)
	if err != nil {
		return nil, err
	}

	dag := merkledag.NewDAGService(bs)
	ns := namesys.NewNameSystem(route)
	p, err := pin.LoadPinner(d, dag)
	if err != nil {
		p = pin.NewPinner(d, dag)
	}

	success = true
	return &IpfsNode{
		Config:      cfg,
		Peerstore:   peerstore,
		Datastore:   d,
		Blocks:      bs,
		DAG:         dag,
		Resolver:    &path.Resolver{DAG: dag},
		Exchange:    exchangeSession,
		Identity:    local,
		Routing:     route,
		Namesys:     ns,
		Diagnostics: diagnostics,
		Network:     network,
		Pinning:     p,
	}, nil
}

func initIdentity(cfg *config.Config, peers peer.Peerstore, online bool) (peer.Peer, error) {
	if cfg.Identity.PeerID == "" {
		return nil, errors.New("Identity was not set in config (was ipfs init run?)")
	}

	if len(cfg.Identity.PeerID) == 0 {
		return nil, errors.New("No peer ID in config! (was ipfs init run?)")
	}

	// get peer from peerstore (so it is constructed there)
	id := peer.ID(b58.Decode(cfg.Identity.PeerID))
	peer, err := peers.Get(id)
	if err != nil {
		return nil, err
	}

	// address is optional
	if len(cfg.Addresses.Swarm) > 0 {
		maddr, err := ma.NewMultiaddr(cfg.Addresses.Swarm)
		if err != nil {
			return nil, err
		}

		peer.AddAddress(maddr)
	}

	// when not online, don't need to parse private keys (yet)
	if online {
		skb, err := base64.StdEncoding.DecodeString(cfg.Identity.PrivKey)
		if err != nil {
			return nil, err
		}

		if err := peer.LoadAndVerifyKeyPair(skb); err != nil {
			return nil, err
		}
	}

	return peer, nil
}

func initConnections(ctx context.Context, cfg *config.Config, pstore peer.Peerstore, route *dht.IpfsDHT) {
	for _, p := range cfg.Bootstrap {
		if p.PeerID == "" {
			log.Errorf("error: peer does not include PeerID. %v", p)
		}

		maddr, err := ma.NewMultiaddr(p.Address)
		if err != nil {
			log.Error(err)
			continue
		}

		// setup peer
		npeer, err := pstore.Get(peer.DecodePrettyID(p.PeerID))
		if err != nil {
			log.Errorf("Bootstrapping error: %v", err)
			continue
		}
		npeer.AddAddress(maddr)

		if _, err = route.Connect(ctx, npeer); err != nil {
			log.Errorf("Bootstrapping error: %v", err)
		}
	}
}

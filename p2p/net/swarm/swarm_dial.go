package swarm

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	conn "github.com/jbenet/go-ipfs/p2p/net/conn"
	addrutil "github.com/jbenet/go-ipfs/p2p/net/swarm/addr"
	peer "github.com/jbenet/go-ipfs/p2p/peer"
	lgbl "github.com/jbenet/go-ipfs/util/eventlog/loggables"

	context "github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"
	ma "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multiaddr-net"
)

// dialAttempts governs how many times a goroutine will try to dial a given peer.
const dialAttempts = 3

// DialTimeout is the amount of time each dial attempt has. We can think about making
// this larger down the road, or putting more granular timeouts (i.e. within each
// subcomponent of Dial)
var DialTimeout time.Duration = time.Second * 10

// dialsync is a small object that helps manage ongoing dials.
// this way, if we receive many simultaneous dial requests, one
// can do its thing, while the rest wait.
//
// this interface is so would-be dialers can just:
//
//  for {
//  	c := findConnectionToPeer(peer)
//  	if c != nil {
//  		return c
//  	}
//
//  	// ok, no connections. should we dial?
//  	if ok, wait := dialsync.Lock(peer); !ok {
//  		<-wait // can optionally wait
//  		continue
//  	}
//  	defer dialsync.Unlock(peer)
//
//  	c := actuallyDial(peer)
//  	return c
//  }
//
type dialsync struct {
	// ongoing is a map of tickets for the current peers being dialed.
	// this way, we dont kick off N dials simultaneously.
	ongoing map[peer.ID]chan struct{}
	lock    sync.Mutex
}

// Lock governs the beginning of a dial attempt.
// If there are no ongoing dials, it returns true, and the client is now
// scheduled to dial. Every other goroutine that calls startDial -- with
//the same dst -- will block until client is done. The client MUST call
// ds.Unlock(p) when it is done, to unblock the other callers.
// The client is not reponsible for achieving a successful dial, only for
// reporting the end of the attempt (calling ds.Unlock(p)).
//
// see the example below `dialsync`
func (ds *dialsync) Lock(dst peer.ID) (bool, chan struct{}) {
	ds.lock.Lock()
	if ds.ongoing == nil { // init if not ready
		ds.ongoing = make(map[peer.ID]chan struct{})
	}
	wait, found := ds.ongoing[dst]
	if !found {
		ds.ongoing[dst] = make(chan struct{})
	}
	ds.lock.Unlock()

	if found {
		return false, wait
	}

	// ok! you're signed up to dial!
	return true, nil
}

// Unlock releases waiters to a dial attempt. see Lock.
// if Unlock(p) is called without calling Lock(p) first, Unlock panics.
func (ds *dialsync) Unlock(dst peer.ID) {
	ds.lock.Lock()
	wait, found := ds.ongoing[dst]
	if !found {
		panic("called dialDone with no ongoing dials to peer: " + dst.Pretty())
	}
	delete(ds.ongoing, dst) // remove ongoing dial
	close(wait)             // release everyone else
	ds.lock.Unlock()
}

// Dial connects to a peer.
//
// The idea is that the client of Swarm does not need to know what network
// the connection will happen over. Swarm can use whichever it choses.
// This allows us to use various transport protocols, do NAT traversal/relay,
// etc. to achive connection.
func (s *Swarm) Dial(ctx context.Context, p peer.ID) (*Conn, error) {
	if p == s.local {
		return nil, errors.New("Attempted connection to self!")
	}

	// this loop is here because dials take time, and we should not be dialing
	// the same peer concurrently (silly waste). Additonally, it's structured
	// to check s.ConnectionsToPeer(p) _first_, and _between_ attempts because we
	// may have received an incoming connection! if so, we no longer must dial.
	//
	// During the dial attempts, we may be doing the dialing. if not, we wait.
	var err error
	var conn *Conn
	for i := 0; i < dialAttempts; i++ {
		// check if we already have an open connection first
		cs := s.ConnectionsToPeer(p)
		for _, conn = range cs {
			if conn != nil { // dump out the first one we find. (TODO pick better)
				return conn, nil
			}
		}

		// check if there's an ongoing dial to this peer
		if ok, wait := s.dsync.Lock(p); !ok {
			log.Debugf("swarm %s dialing %s -- waiting for ongoing dial", s.local, p)
			select {
			case <-wait: // wait for that dial to finish.
				continue // and see if it worked (loop), OR we got an incoming dial.
			case <-ctx.Done(): // or we may have to bail...
				return nil, ctx.Err()
			}
		}

		// ok, we have been charged to dial! let's do it.
		// if it succeeds, dial will add the conn to the swarm itself.
		log.Debugf("swarm %s dialing %s -- dial start", s.local, p)
		ctxT, _ := context.WithTimeout(ctx, DialTimeout)
		conn, err = s.dial(ctxT, p)
		s.dsync.Unlock(p)
		log.Debugf("swarm %s dialing %s -- dial end %s", s.local, p, conn)
		if err != nil {
			continue // ok, we failed. try again. (if loop is done, our error is output)
		}
		return conn, nil
	}
	if err == nil {
		err = fmt.Errorf("%s failed to dial %s after %d attempts", s.local, p, dialAttempts)
	}
	return nil, err
}

// dial is the actual swarm's dial logic, gated by Dial.
func (s *Swarm) dial(ctx context.Context, p peer.ID) (*Conn, error) {
	if p == s.local {
		return nil, errors.New("Attempted connection to self!")
	}

	sk := s.peers.PrivKey(s.local)
	if sk == nil {
		// may be fine for sk to be nil, just log a warning.
		log.Warning("Dial not given PrivateKey, so WILL NOT SECURE conn.")
	}

	// get our own addrs
	localAddrs := s.peers.Addresses(s.local)
	if len(localAddrs) == 0 {
		log.Debug("Dialing out with no local addresses.")
	}

	// get remote peer addrs
	remoteAddrs := s.peers.Addresses(p)
	// make sure we can use the addresses.
	remoteAddrs = addrutil.FilterUsableAddrs(remoteAddrs)
	// drop out any addrs that would just dial ourselves. use ListenAddresses
	// as that is a more authoritative view than localAddrs.
	ila, _ := InterfaceListenAddresses(s)
	remoteAddrs = addrutil.Subtract(remoteAddrs, ila)
	remoteAddrs = addrutil.Subtract(remoteAddrs, s.peers.Addresses(s.local))
	log.Debugf("%s swarm dialing %s -- remote:%s local:%s", s.local, p, remoteAddrs, s.ListenAddresses())
	if len(remoteAddrs) == 0 {
		return nil, errors.New("peer has no addresses")
	}

	// open connection to peer
	d := &conn.Dialer{
		Dialer: manet.Dialer{
			Dialer: net.Dialer{
				Timeout: DialTimeout,
			},
		},
		LocalPeer:  s.local,
		LocalAddrs: localAddrs,
		PrivateKey: sk,
	}

	// try to get a connection to any addr
	connC, err := s.dialAddrs(ctx, d, p, remoteAddrs)
	if err != nil {
		return nil, err
	}

	// ok try to setup the new connection.
	swarmC, err := dialConnSetup(ctx, s, connC)
	if err != nil {
		log.Error("Dial newConnSetup failed. disconnecting.")
		log.Event(ctx, "dialFailureDisconnect", lgbl.NetConn(connC), lgbl.Error(err))
		connC.Close() // close the connection. didn't work out :(
		return nil, err
	}

	log.Event(ctx, "dial", p)
	return swarmC, nil
}

func (s *Swarm) dialAddrs(ctx context.Context, d *conn.Dialer, p peer.ID, remoteAddrs []ma.Multiaddr) (conn.Conn, error) {

	// try to connect to one of the peer's known addresses.
	// for simplicity, we do this sequentially.
	// A future commit will do this asynchronously.
	log.Debugf("%s swarm dialing %s %s", s.local, p, remoteAddrs)
	var err error
	for _, addr := range remoteAddrs {
		log.Debugf("%s swarm dialing %s %s", s.local, p, addr)
		var connC conn.Conn
		connC, err = d.Dial(ctx, addr, p)
		if err != nil {
			log.Info("%s --> %s dial attempt failed: %s", s.local, p, err)
			continue
		}

		// if the connection is not to whom we thought it would be...
		if connC.RemotePeer() != p {
			log.Infof("misdial to %s through %s (got %s)", p, addr, connC.RemotePeer())
			connC.Close()
			continue
		}

		// if the connection is to ourselves...
		// this can happen TONS when Loopback addrs are advertized.
		// (this should be caught by two checks above, but let's just make sure.)
		if connC.RemotePeer() == s.local {
			log.Infof("misdial to %s through %s", p, addr)
			connC.Close()
			continue
		}

		// success! we got one!
		return connC, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("failed to dial %s", p)
}

// dialConnSetup is the setup logic for a connection from the dial side. it
// needs to add the Conn to the StreamSwarm, then run newConnSetup
func dialConnSetup(ctx context.Context, s *Swarm, connC conn.Conn) (*Conn, error) {

	psC, err := s.swarm.AddConn(connC)
	if err != nil {
		// connC is closed by caller if we fail.
		return nil, fmt.Errorf("failed to add conn to ps.Swarm: %s", err)
	}

	// ok try to setup the new connection. (newConnSetup will add to group)
	swarmC, err := s.newConnSetup(ctx, psC)
	if err != nil {
		log.Error("Dial newConnSetup failed. disconnecting.")
		log.Event(ctx, "dialFailureDisconnect", lgbl.NetConn(connC), lgbl.Error(err))
		psC.Close() // we need to make sure psC is Closed.
		return nil, err
	}

	return swarmC, err
}

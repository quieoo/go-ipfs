package config

import (
	"encoding/base64"
	ic "gx/ipfs/QmNefBbWHR9JEiP3KDVqZsBLQVRmH3GBG2D2Ke24SsFqfW/go-libp2p/p2p/crypto"
)

// Identity tracks the configuration of the local node's identity.
type Identity struct {
	PeerID  string
	PrivKey string
}

// DecodePrivateKey is a helper to decode the users PrivateKey
func (i *Identity) DecodePrivateKey(passphrase string) (ic.PrivKey, error) {
	pkb, err := base64.StdEncoding.DecodeString(i.PrivKey)
	if err != nil {
		return nil, err
	}

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	return ic.UnmarshalPrivateKey(pkb)
}

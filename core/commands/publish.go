package commands

import (
	"errors"
	"fmt"
	"io"

	"github.com/jbenet/go-ipfs/core"
	u "github.com/jbenet/go-ipfs/util"
	"github.com/op/go-logging"

	nsys "github.com/jbenet/go-ipfs/namesys"
)

var log = logging.MustGetLogger("commands")

func Publish(n *core.IpfsNode, args []string, opts map[string]interface{}, out io.Writer) error {
	log.Debug("Begin Publish")
	if n.Identity == nil {
		return errors.New("Identity not loaded!")
	}

	k := n.Identity.PrivKey
	val := u.Key(args[0])

	pub := nsys.NewPublisher(n.DAG, n.Routing)
	err := pub.Publish(k, val)
	if err != nil {
		return err
	}

	hash, err := k.GetPublic().Hash()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Published %s to %s\n", val, u.Key(hash).Pretty())

	return nil
}

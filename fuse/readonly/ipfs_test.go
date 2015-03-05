// +build !nofuse

package readonly

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"

	fstest "github.com/jbenet/go-ipfs/Godeps/_workspace/src/bazil.org/fuse/fs/fstestutil"

	core "github.com/jbenet/go-ipfs/core"
	coreunix "github.com/jbenet/go-ipfs/core/coreunix"
	importer "github.com/jbenet/go-ipfs/importer"
	chunk "github.com/jbenet/go-ipfs/importer/chunk"
	dag "github.com/jbenet/go-ipfs/merkledag"
	uio "github.com/jbenet/go-ipfs/unixfs/io"
	u "github.com/jbenet/go-ipfs/util"
	ci "github.com/jbenet/go-ipfs/util/testutil/ci"
)

func maybeSkipFuseTests(t *testing.T) {
	if ci.NoFuse() {
		t.Skip("Skipping FUSE tests")
	}
}

func randObj(t *testing.T, nd *core.IpfsNode, size int64) (*dag.Node, []byte) {
	buf := make([]byte, size)
	u.NewTimeSeededRand().Read(buf)
	read := bytes.NewReader(buf)
	obj, err := importer.BuildTrickleDagFromReader(read, nd.DAG, nil, chunk.DefaultSplitter)
	if err != nil {
		t.Fatal(err)
	}

	return obj, buf
}

func setupIpfsTest(t *testing.T, node *core.IpfsNode) (*core.IpfsNode, *fstest.Mount) {
	maybeSkipFuseTests(t)

	var err error
	if node == nil {
		node, err = core.NewMockNode()
		if err != nil {
			t.Fatal(err)
		}
	}

	fs := NewFileSystem(node)
	mnt, err := fstest.MountedT(t, fs)
	if err != nil {
		t.Fatal(err)
	}

	return node, mnt
}

// Test writing an object and reading it back through fuse
func TestIpfsBasicRead(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	nd, mnt := setupIpfsTest(t, nil)
	defer mnt.Close()

	fi, data := randObj(t, nd, 10000)
	k, err := fi.Key()
	if err != nil {
		t.Fatal(err)
	}

	fname := path.Join(mnt.Dir, k.String())
	rbuf, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(rbuf, data) {
		t.Fatal("Incorrect Read!")
	}
}

func getPaths(t *testing.T, ipfs *core.IpfsNode, name string, n *dag.Node) []string {
	if len(n.Links) == 0 {
		return []string{name}
	}
	var out []string
	for _, lnk := range n.Links {
		child, err := lnk.GetNode(ipfs.DAG)
		if err != nil {
			t.Fatal(err)
		}
		sub := getPaths(t, ipfs, path.Join(name, lnk.Name), child)
		out = append(out, sub...)
	}
	return out
}

// Perform a large number of concurrent reads to stress the system
func TestIpfsStressRead(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	nd, mnt := setupIpfsTest(t, nil)
	defer mnt.Close()

	var ks []u.Key
	var paths []string

	nobj := 50
	ndiriter := 50

	// Make a bunch of objects
	for i := 0; i < nobj; i++ {
		fi, _ := randObj(t, nd, rand.Int63n(50000))
		k, err := fi.Key()
		if err != nil {
			t.Fatal(err)
		}

		ks = append(ks, k)
		paths = append(paths, k.String())
	}

	// Now make a bunch of dirs
	for i := 0; i < ndiriter; i++ {
		db := uio.NewDirectory(nd.DAG)
		for j := 0; j < 1+rand.Intn(10); j++ {
			name := fmt.Sprintf("child%d", j)
			err := db.AddChild(name, ks[rand.Intn(len(ks))])
			if err != nil {
				t.Fatal(err)
			}
		}
		newdir := db.GetNode()
		k, err := nd.DAG.Add(newdir)
		if err != nil {
			t.Fatal(err)
		}

		ks = append(ks, k)
		npaths := getPaths(t, nd, k.String(), newdir)
		paths = append(paths, npaths...)
	}

	// Now read a bunch, concurrently
	wg := sync.WaitGroup{}

	for s := 0; s < 4; s++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 2000; i++ {
				item := paths[rand.Intn(len(paths))]
				fname := path.Join(mnt.Dir, item)
				rbuf, err := ioutil.ReadFile(fname)
				if err != nil {
					t.Fatal(err)
				}

				read, err := coreunix.Cat(nd, item)
				if err != nil {
					t.Fatal(err)
				}

				data, err := ioutil.ReadAll(read)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(rbuf, data) {
					t.Fatal("Incorrect Read!")
				}
			}
		}()
	}
	wg.Wait()
}

// Test writing a file and reading it back
func TestIpfsBasicDirRead(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	nd, mnt := setupIpfsTest(t, nil)
	defer mnt.Close()

	// Make a 'file'
	fi, data := randObj(t, nd, 10000)
	k, err := fi.Key()
	if err != nil {
		t.Fatal(err)
	}

	// Make a directory and put that file in it
	db := uio.NewDirectory(nd.DAG)
	err = db.AddChild("actual", k)
	if err != nil {
		t.Fatal(err)
	}

	d1nd := db.GetNode()
	d1ndk, err := nd.DAG.Add(d1nd)
	if err != nil {
		t.Fatal(err)
	}

	dirname := path.Join(mnt.Dir, d1ndk.String())
	fname := path.Join(dirname, "actual")
	rbuf, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Fatal(err)
	}

	dirents, err := ioutil.ReadDir(dirname)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirents) != 1 {
		t.Fatal("Bad directory entry count")
	}
	if dirents[0].Name() != "actual" {
		t.Fatal("Bad directory entry")
	}

	if !bytes.Equal(rbuf, data) {
		t.Fatal("Incorrect Read!")
	}
}

// Test to make sure the filesystem reports file sizes correctly
func TestFileSizeReporting(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	nd, mnt := setupIpfsTest(t, nil)
	defer mnt.Close()

	fi, data := randObj(t, nd, 10000)
	k, err := fi.Key()
	if err != nil {
		t.Fatal(err)
	}

	fname := path.Join(mnt.Dir, k.String())

	finfo, err := os.Stat(fname)
	if err != nil {
		t.Fatal(err)
	}

	if finfo.Size() != int64(len(data)) {
		t.Fatal("Read incorrect size from stat!")
	}
}

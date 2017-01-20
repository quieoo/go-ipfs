package mfsr

import (
	"io/ioutil"
	"testing"

	"github.com/ipfs/go-ipfs/thirdparty/assert"
)

func testVersionFile(v string, t *testing.T) (rp RepoPath) {
	name, err := ioutil.TempDir("", v)
	if err != nil {
		t.Fatal(err)
	}
	rp = RepoPath(name)
	return rp
}

func TestVersion(t *testing.T) {
	rp := RepoPath("")
	_, err := rp.Version()
	assert.Err(err, t, "Should throw an error when path is bad,")

	rp = testVersionFile("4", t)
	_, err = rp.Version()
	assert.Err(err, t, "Bad VersionFile")

	assert.Nil(rp.WriteVersion(4), t, "Trouble writing version")

	assert.Nil(rp.CheckVersion(4), t, "Trouble checking the verion")

	assert.Err(rp.CheckVersion(1), t, "Should throw an error for the wrong version.")
}

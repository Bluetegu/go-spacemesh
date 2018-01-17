package test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/spacemeshos/go-spacemesh/assert"
	"github.com/spacemeshos/go-spacemesh/crypto"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/merkle"
)

func TestEmptyTreeCreation(t *testing.T) {

	userDb, treeDb := getDbPaths(t)
	m, err := merkle.NewEmptyTree(userDb, treeDb)
	assert.NoErr(t, err, "failed to create new merkle tree")

	root := m.GetRootNode()
	assert.Nil(t, root, "expected empty tree")

	hash, err := m.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")
	assert.True(t, bytes.Equal(merkle.EmptyTreeRootHash, hash), "unexpected empty tree root hash")

	err = m.CloseDataStores()
	assert.NoErr(t, err, "failed to close data stores")
}

// Test a simple 1-node merkle tree
func TestSimpleTreeOps(t *testing.T) {

	userDb, treeDb := getDbPaths(t)
	m, err := merkle.NewEmptyTree(userDb, treeDb)
	defer m.CloseDataStores() // we need to close the data stores when done w m - they are owned by m

	assert.NoErr(t, err, "failed to create new merkle tree")

	// user data k,v can be any bytes
	v := []byte("zifton-the-immortal")
	k := []byte("the-name-of-my-cat")

	log.Info("User key hex: %s", hex.EncodeToString(k))

	tryPut(t, m, k, v)

	root := m.GetRootNode()
	assert.NotNil(t, root, "expected non-empty tree")

	validateGet(t, m, k, v)

	err = m.CloseDataStores()
	assert.NoErr(t, err, "failed to close m data stores")

	/////////////////////////

	// restore tree to a new instance based on root hash
	rootHash, err := m.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")
	m1, _ := merkle.NewTreeFromDb(rootHash, userDb, treeDb)
	defer m1.CloseDataStores() // tell m1 to close data stores when we are done w it

	root = m1.GetRootNode()
	assert.NotNil(t, root, "expected non-empty tree")

	rootHash1, err := m1.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")

	assert.True(t, bytes.Equal(rootHash, rootHash1), "expected same root hash")

	// test getting the data from the new tree instance

	validateGet(t, m1, k, v)

}

// Test a simple 1-node merkle tree
func TestComplexTreeOps(t *testing.T) {

	k1, err := hex.DecodeString("123456")
	assert.NoErr(t, err, "invalid hex str")
	v1 := []byte("zifton")

	k2, err := hex.DecodeString("112456")
	assert.NoErr(t, err, "invalid hex str")
	v2 := []byte("tantalus")

	// ext, path: 1, key: branch
	// branch
	//  [1] -> leaf (2456,v)
	//	[2] -> leaf (3456,v)
	//

	k3, err := hex.DecodeString("112457")
	assert.NoErr(t, err, "invalid hex str")
	v3, err := crypto.GetRandomBytes(100)
	assert.NoErr(t, err, "failed to get random data")

	k4, err := hex.DecodeString("123457")
	assert.NoErr(t, err, "invalid hex str")
	v4, err := crypto.GetRandomBytes(100)
	assert.NoErr(t, err, "failed to get random data")

	userDb, treeDb := getDbPaths(t)
	m, err := merkle.NewEmptyTree(userDb, treeDb)
	assert.NoErr(t, err, "failed to create new Merkle tree")
	defer m.CloseDataStores() // we need to close the data stores when done w m - they are owned by m

	tryPut(t, m, k1, v1)

	r, err := m.ValidateStructure(m.GetRootNode())
	assert.NoErr(t, err, "invalid tree structure")
	mHash, err := m.GetRootHash()
	assert.NoErr(t, err, "invalid tree structure")
	assert.True(t, bytes.Equal(r, mHash), "unexpected root hash")

	log.Info(m.Print())
	validateGet(t, m, k1, v1)

	tryPut(t, m, k2, v2)

	log.Info(m.Print())
	validateGet(t, m, k1, v1)
	validateGet(t, m, k2, v2)

	r, err = m.ValidateStructure(m.GetRootNode())
	assert.NoErr(t, err, "invalid tree structure")
	mHash, err = m.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")
	assert.True(t, bytes.Equal(r, mHash), "unexpected root hash")

	data, _, err := m.Get(k3)
	assert.True(t, len(data) == 0, "expected empty result")
	assert.NoErr(t, err, "expected no error")

	tryPut(t, m, k3, v3)

	// expected structure:
	//
	// root: ext, 1
	//   branch
	//     [1] -> -> ext(245) -> branch
	// 								[6] -> (<>,v)
	//								[7] -> (<>,v)
	//	 [2] -> leaf (3456,v)
	//

	//1 12457
	//1 12456

	log.Info(m.Print())
	r, err = m.ValidateStructure(m.GetRootNode())
	assert.NoErr(t, err, "invalid tree structure")
	mHash, err = m.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")
	assert.True(t, bytes.Equal(r, mHash), "unexpected root hash")

	validateGet(t, m, k1, v1)
	validateGet(t, m, k2, v2)
	validateGet(t, m, k3, v3)

	// key 123457

	tryPut(t, m, k4, v4)
	log.Info(m.Print())
	r, err = m.ValidateStructure(m.GetRootNode())
	assert.NoErr(t, err, "invalid tree structure")
	mHash, err = m.GetRootHash()
	assert.NoErr(t, err, "error getting node hash")
	assert.True(t, bytes.Equal(r, mHash), "unexpected root hash")

	validateGet(t, m, k1, v1)
	validateGet(t, m, k2, v2)
	validateGet(t, m, k3, v3)
	validateGet(t, m, k4, v4)

	// expected structure:
	// 123456
	// 112456
	// 112457
	// 123457
	//
	// root: ext, 1
	//   branch
	//     [1] -> -> ext(245) -> branch
	// 								[6] -> (<>,v)
	//								[7] -> (<>,v)
	//	   [2] -> ext(345) -> branch
	// 								[6] leaf (<>,v)
	//								[7] leaf (<>,v)

}

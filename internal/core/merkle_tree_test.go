package core

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMerkleTree(t *testing.T) {
	testCases := []struct {
		name          string
		data          [][]byte
		expectedError bool
		expectedLeafs int
	}{
		{"Valid data", [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")}, false, 4},
		{"Empty data", [][]byte{}, true, 0},
		{"Single transaction", [][]byte{[]byte("tx1")}, false, 2},
		{"Power of two transactions", [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3"), []byte("tx4")}, false, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := NewMerkleTree(tc.data)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tree.Root)
				assert.Len(t, tree.Leafs, tc.expectedLeafs)
			}
		})
	}
}

func TestGetRoot(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2")}
	tree, err := NewMerkleTree(data)
	require.NoError(t, err)

	root := tree.GetRoot()
	assert.NotEmpty(t, root)
	assert.Len(t, root, 128) // SHA-512 hash is 64 bytes, hex-encoded is 128 characters
}

func TestGetProof(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3"), []byte("tx4")}
	tree, err := NewMerkleTree(data)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		index         int
		expectedError bool
		expectedProof int
	}{
		{"Valid index", 2, false, 2},
		{"Out of range index", 4, true, 0},
		{"Negative index", -1, true, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proof, err := tree.GetProof(tc.index)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, proof, tc.expectedProof)
			}
		})
	}
}

func TestVerifyProof(t *testing.T) {
	data := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3"), []byte("tx4")}
	tree, err := NewMerkleTree(data)
	require.NoError(t, err)
	root := tree.GetRoot()

	for i, datum := range data {
		proof, err := tree.GetProof(i)
		require.NoError(t, err)
		hash := sha512.Sum512(datum)
		leaf := hex.EncodeToString(hash[:])
		result, err := VerifyProof(leaf, proof, root)
		assert.NoError(t, err)
		assert.True(t, result)
	}

	// Test invalid proof
	invalidProof := []string{"invalidhash"}
	hash := sha512.Sum512(data[0])
	result, err := VerifyProof(hex.EncodeToString(hash[:]), invalidProof, root)
	assert.Error(t, err)
	assert.False(t, result)

	// Test valid proof for a different tree
	otherData := [][]byte{[]byte("tx5"), []byte("tx6")}
	otherTree, _ := NewMerkleTree(otherData)
	otherRoot := otherTree.GetRoot()
	proof, _ := tree.GetProof(0)
	result, err = VerifyProof(hex.EncodeToString(hash[:]), proof, otherRoot)
	assert.NoError(t, err)
	assert.False(t, result)
}

func TestMerkleTreeSecurity(t *testing.T) {
	// Test against second preimage attack
	data1 := [][]byte{[]byte("tx1"), []byte("tx2")}
	data2 := [][]byte{[]byte("tx3"), []byte("tx4")}

	tree1, err := NewMerkleTree(data1)
	assert.NoError(t, err)
	tree2, err := NewMerkleTree(data2)
	assert.NoError(t, err)
	assert.NotEqual(t, tree1.GetRoot(), tree2.GetRoot())

	// Test against collision resistance
	collisionData := [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx1"), []byte("tx2")}
	collisionTree, err := NewMerkleTree(collisionData)
	assert.NoError(t, err)
	proof1, err := collisionTree.GetProof(0)
	assert.NoError(t, err)
	proof3, err := collisionTree.GetProof(2)
	assert.NoError(t, err)

	assert.NotEqual(t, proof1, proof3)
}

func BenchmarkNewMerkleTree(b *testing.B) {
	data := make([][]byte, 1000)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("tx%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewMerkleTree(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestNewMerkleTreeEdgeCases(t *testing.T) {
	// Test with single data item
	data := [][]byte{[]byte("Single Transaction")}
	tree, err := NewMerkleTree(data)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Len(t, tree.Leafs, 2) // Original + duplicate

	// Test with very large number of transactions
	largeData := make([][]byte, 1000000)
	for i := range largeData {
		largeData[i] = []byte(fmt.Sprintf("tx%d", i))
	}
	tree, err = NewMerkleTree(largeData)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
}

func FuzzMerkleTree(f *testing.F) {
	f.Add([]byte("tx1"), []byte("tx2"))
	f.Fuzz(func(t *testing.T, a, b []byte) {
		data := [][]byte{a, b}
		tree, err := NewMerkleTree(data)
		if err != nil {
			return
		}
		root := tree.GetRoot()
		proof, err := tree.GetProof(0)
		if err != nil {
			return
		}
		hash := sha512.Sum512(a)
		result, err := VerifyProof(hex.EncodeToString(hash[:]), proof, root)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

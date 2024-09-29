package core

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

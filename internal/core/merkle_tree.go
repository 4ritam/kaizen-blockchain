package core

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	Root   *MerkleNode
	Leafs  []*MerkleNode
	levels [][]*MerkleNode
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Left     *MerkleNode
	Right    *MerkleNode
	Hash     []byte
	Position int
}

// NewMerkleTree creates a new Merkle tree from a list of data
func NewMerkleTree(data interface{}) (*MerkleTree, error) {
	dataBytes, ok := data.([][]byte)
	if !ok {
		return nil, errors.New("invalid data type: expected [][]byte")
	}
	if len(dataBytes) == 0 {
		return nil, errors.New("cannot create Merkle tree with no data")
	}

	return newMerkleTree(dataBytes)
}

func newMerkleTree(data [][]byte) (*MerkleTree, error) {
	var leafs []*MerkleNode
	for _, datum := range data {
		hash := sha512.Sum512(datum)
		leafs = append(leafs, &MerkleNode{Hash: hash[:]})
	}

	if len(leafs)%2 == 1 {
		secondHash := sha512.Sum512(append(leafs[0].Hash, byte(0)))
		leafs = append(leafs, &MerkleNode{Hash: secondHash[:]})
	}

	tree := &MerkleTree{
		Leafs:  leafs,
		levels: [][]*MerkleNode{leafs},
	}

	tree.Root = tree.buildTree()

	return tree, nil
}

func (m *MerkleTree) buildTree() *MerkleNode {
	for level := 0; len(m.levels[len(m.levels)-1]) > 1; level++ {
		var currentLevel []*MerkleNode
		for i := 0; i < len(m.levels[len(m.levels)-1]); i += 2 {
			left := m.levels[len(m.levels)-1][i]
			var right *MerkleNode
			if i+1 < len(m.levels[len(m.levels)-1]) {
				right = m.levels[len(m.levels)-1][i+1]
			} else {
				right = left
			}
			combinedHash := append(left.Hash, right.Hash...)
			positionBytes := []byte(fmt.Sprintf("%d:%d", level, i/2))
			hash := sha512.Sum512(append(combinedHash, positionBytes...))
			parent := &MerkleNode{
				Left:     left,
				Right:    right,
				Hash:     hash[:],
				Position: i / 2,
			}
			currentLevel = append(currentLevel, parent)
		}
		m.levels = append(m.levels, currentLevel)
	}
	return m.levels[len(m.levels)-1][0]
}

// GetRoot returns the Merkle root hash as a hexadecimal string
func (m *MerkleTree) GetRoot() string {
	return hex.EncodeToString(m.Root.Hash)
}

// GetProof generates a Merkle proof for a given leaf index
func (m *MerkleTree) GetProof(index interface{}) ([]string, error) {
	idx, ok := index.(int)
	if !ok {
		return nil, errors.New("invalid index type: expected int")
	}
	if idx < 0 || idx >= len(m.Leafs) {
		return nil, errors.New("leaf index out of range")
	}

	return m.getProof(idx)
}

func (m *MerkleTree) getProof(index int) ([]string, error) {
	if index < 0 || index >= len(m.Leafs) {
		return nil, errors.New("leaf index out of range")
	}

	var proof []string
	currentIndex := index

	for i := 0; i < len(m.levels)-1; i++ {
		levelLen := len(m.levels[i])
		if currentIndex%2 == 0 {
			if currentIndex+1 < levelLen {
				sibling := m.levels[i][currentIndex+1]
				proof = append(proof, fmt.Sprintf("%d:%d:%s", i, currentIndex+1, hex.EncodeToString(sibling.Hash)))
			}
		} else {
			sibling := m.levels[i][currentIndex-1]
			proof = append(proof, fmt.Sprintf("%d:%d:%s", i, currentIndex-1, hex.EncodeToString(sibling.Hash)))
		}
		currentIndex /= 2
	}

	return proof, nil
}

// VerifyProof verifies a Merkle proof
func VerifyProof(leaf interface{}, proof interface{}, root interface{}) (bool, error) {
	leafStr, ok := leaf.(string)
	if !ok {
		return false, errors.New("invalid leaf type: expected string")
	}
	proofSlice, ok := proof.([]string)
	if !ok {
		return false, errors.New("invalid proof type: expected []string")
	}
	rootStr, ok := root.(string)
	if !ok {
		return false, errors.New("invalid root type: expected string")
	}

	return verifyProof(leafStr, proofSlice, rootStr)
}

func verifyProof(leaf string, proof []string, root string) (bool, error) {
    currentHash, err := hex.DecodeString(leaf)
    if err != nil {
        return false, fmt.Errorf("invalid leaf hash: %v", err)
    }

    for _, proofElement := range proof {
        parts := strings.Split(proofElement, ":")
        if len(parts) != 3 {
            return false, fmt.Errorf("invalid proof element format")
        }
        level, err := strconv.Atoi(parts[0])
        if err != nil {
            return false, fmt.Errorf("invalid level in proof element: %v", err)
        }
        position, err := strconv.Atoi(parts[1])
        if err != nil {
            return false, fmt.Errorf("invalid position in proof element: %v", err)
        }
        proofHash, err := hex.DecodeString(parts[2])
        if err != nil {
            return false, fmt.Errorf("invalid hash in proof element: %v", err)
        }

        var combinedHash []byte
        if position%2 == 0 {
            combinedHash = append(proofHash, currentHash...)
        } else {
            combinedHash = append(currentHash, proofHash...)
        }
        positionBytes := []byte(fmt.Sprintf("%d:%d", level, position/2))
        hash := sha512.Sum512(append(combinedHash, positionBytes...))
        currentHash = hash[:]
    }

    return hex.EncodeToString(currentHash) == root, nil
}

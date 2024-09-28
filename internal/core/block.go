package core

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/4ritam/kaizen-blockchain/internal/crypto"
)

// BlockVersion represents the version of the block structure
type BlockVersion uint32

// BlockHeight represents the height of the block in the blockchain
type BlockHeight uint64

// BlockHash represents a hash value in the blockchain
type BlockHash string

// BlockNonce represents the nonce value used in mining
type BlockNonce uint64

// DataType represents the type of data stored in a block
type DataType string

const (
	DataTypeTransaction   DataType = "transaction"
	DataTypeDocument      DataType = "document"
	DataTypeSmartContract DataType = "smart_contract"
	// Add more data types as needed
)

// Data represents a piece of data stored in the block
type Data struct {
	Type    DataType    `json:"type"`
	Content interface{} `json:"content"`
}

// Block represents a block in the blockchain
type Block struct {
	Version       BlockVersion `json:"version"`
	Height        BlockHeight  `json:"height"`
	Timestamp     time.Time    `json:"timestamp"`
	PrevBlockHash BlockHash    `json:"prev_block_hash"`
	Hash          BlockHash    `json:"hash"`
	MerkleRoot    BlockHash    `json:"merkle_root"`
	Data          []Data       `json:"data"`
	Nonce         BlockNonce   `json:"nonce"`
	mu            sync.RWMutex
}

// NewBlock creates a new Block with the given parameters after sanitizing inputs
func NewBlock(version interface{}, height interface{}, prevBlockHash interface{}, data interface{}) (*Block, error) {
	// Input sanitization
	v, err := toUint32(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version: %v", err)
	}

	h, err := toUint64(height)
	if err != nil {
		return nil, fmt.Errorf("invalid height: %v", err)
	}

	pbh, ok := prevBlockHash.(string)
	if !ok {
		return nil, fmt.Errorf("invalid previous block hash type")
	}
	if pbh == "" {
		return nil, fmt.Errorf("previous block hash cannot be empty")
	}

	var d []Data
	if data != nil {
		d, ok = data.([]Data)
		if !ok {
			return nil, fmt.Errorf("invalid data type")
		}
	} else {
		d = []Data{} // Initialize to empty slice if nil
	}

	return newBlock(BlockVersion(v), BlockHeight(h), BlockHash(pbh), d)
}

// newBlock is a private function that creates a new Block without input validation
func newBlock(version BlockVersion, height BlockHeight, prevBlockHash BlockHash, data []Data) (*Block, error) {
	block := &Block{
		Version:       version,
		Height:        height,
		Timestamp:     time.Now().UTC(),
		PrevBlockHash: prevBlockHash,
		Data:          data,
	}

	var err error
	block.MerkleRoot, err = block.calculateMerkleRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate merkle root: %v", err)
	}

	block.Hash, err = block.CalculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate block hash: %v", err)
	}

	return block, nil
}

// calculateMerkleRoot calculates the Merkle root of the block's data
func (b *Block) calculateMerkleRoot() (BlockHash, error) {
	if len(b.Data) == 0 {
		// Return a special hash for empty data
		return BlockHash(hex.EncodeToString(make([]byte, 64))), nil
	}

	var dataBytes [][]byte
	for _, d := range b.Data {
		jsonData, err := json.Marshal(d)
		if err != nil {
			return "", fmt.Errorf("failed to marshal data: %v", err)
		}
		dataBytes = append(dataBytes, jsonData)
	}

	merkleTree, err := NewMerkleTree(dataBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create merkle tree: %v", err)
	}
	return BlockHash(merkleTree.GetRoot()), nil
}

// CalculateHash generates the SHA-512 hash of the block
func (b *Block) CalculateHash() (BlockHash, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.calculateHash()
}

func (b *Block) calculateHash() (BlockHash, error) {
	blockData, err := json.Marshal(struct {
		Version       BlockVersion
		Height        BlockHeight
		Timestamp     time.Time
		PrevBlockHash BlockHash
		MerkleRoot    BlockHash
		Data          []Data
		Nonce         BlockNonce
	}{
		Version:       b.Version,
		Height:        b.Height,
		Timestamp:     b.Timestamp,
		PrevBlockHash: b.PrevBlockHash,
		MerkleRoot:    b.MerkleRoot,
		Data:          b.Data,
		Nonce:         b.Nonce,
	})

	if err != nil {
		return "", fmt.Errorf("failed to marshal block data: %v", err)
	}

	hash := sha512.Sum512(blockData)
	return BlockHash(hex.EncodeToString(hash[:])), nil
}

// Validate checks if the block is valid
func (b *Block) Validate() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.validate()
}

func (b *Block) validate() error {
	hash, err := b.calculateHash()
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %v", err)
	}

	if b.Hash != hash {
		return fmt.Errorf("invalid block hash")
	}

	if !crypto.ValidateHash(string(b.Hash)) {
		return fmt.Errorf("invalid hash format")
	}

	if b.Timestamp.After(time.Now().UTC()) {
		return fmt.Errorf("block timestamp is in the future")
	}

	merkleroot, err := b.calculateMerkleRoot()
	if err != nil {
		return fmt.Errorf("failed to calculate Merkle root: %v", err)
	}

	if b.MerkleRoot != merkleroot {
		return fmt.Errorf("invalid Merkle root")
	}

	return nil
}
func (b *Block) AddData(dataType interface{}, content interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	dt, ok := dataType.(DataType)
	if !ok {
		return fmt.Errorf("invalid data type")
	}
	if !isValidDataType(dt) {
		return fmt.Errorf("unsupported data type: %s", dt)
	}
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	return b.addData(dt, content)
}

func (b *Block) addData(dataType DataType, content interface{}) error {
	b.Data = append(b.Data, Data{Type: dataType, Content: content})

	var err error
	b.MerkleRoot, err = b.calculateMerkleRoot()
	if err != nil {
		return fmt.Errorf("failed to recalculate merkle root: %v", err)
	}

	b.Hash, err = b.calculateHash()
	if err != nil {
		return fmt.Errorf("failed to recalculate block hash: %v", err)
	}

	return nil
}

func isValidDataType(dataType DataType) bool {
	switch dataType {
	case DataTypeTransaction, DataTypeDocument, DataTypeSmartContract:
		return true
	default:
		return false
	}
}

func toUint32(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint32(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint32(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint32(v), nil
	case uint:
		return uint32(v), nil
	case uint32:
		return v, nil
	case uint64:
		if v > math.MaxUint32 {
			return 0, fmt.Errorf("value exceeds maximum for uint32")
		}
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("unsupported type for conversion to uint32")
	}
}

func toUint64(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value not allowed")
		}
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	default:
		return 0, fmt.Errorf("unsupported type for conversion to uint64")
	}
}

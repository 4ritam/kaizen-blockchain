package core

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testVersion       = uint32(1)
	testHeight        = uint64(1)
	testPrevBlockHash = "prevHash"
)

func TestNewBlock(t *testing.T) {
	data := []Data{
		{Type: DataTypeTransaction, Content: "test transaction"},
	}
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, data)
	require.NoError(t, err)
	assert.NotNil(t, block)

	assert.Equal(t, BlockVersion(testVersion), block.Version)
	assert.Equal(t, BlockHeight(testHeight), block.Height)
	assert.Equal(t, BlockHash(testPrevBlockHash), block.PrevBlockHash)
	assert.Equal(t, data, block.Data)
	assert.NotEmpty(t, block.Hash)
	assert.NotEmpty(t, block.MerkleRoot)
	assert.False(t, block.Timestamp.After(time.Now().UTC()))
	assert.Equal(t, BlockNonce(0), block.Nonce)
}

func TestCalculateHash(t *testing.T) {
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, []Data{})
	require.NoError(t, err)

	hash1, err := block.CalculateHash()
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	block.Nonce++
	hash2, err := block.CalculateHash()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)

	block.Data = append(block.Data, Data{Type: DataTypeTransaction, Content: "new tx"})
	hash3, err := block.CalculateHash()
	require.NoError(t, err)
	assert.NotEqual(t, hash2, hash3)
}

func TestValidate(t *testing.T) {
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, []Data{})
	require.NoError(t, err)

	assert.NoError(t, block.Validate())

	// Test invalid hash
	originalHash := block.Hash
	block.Hash = "invalidHash"
	assert.Error(t, block.Validate())
	block.Hash = originalHash

	// Test future timestamp
	originalTimestamp := block.Timestamp
	block.Timestamp = time.Now().Add(1 * time.Hour)
	assert.Error(t, block.Validate())
	block.Timestamp = originalTimestamp

	// Test invalid Merkle root
	originalRoot := block.MerkleRoot
	block.MerkleRoot = "invalidRoot"
	assert.Error(t, block.Validate())
	block.MerkleRoot = originalRoot
}

func TestAddData(t *testing.T) {
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, []Data{})
	require.NoError(t, err)

	originalHash := block.Hash
	originalRoot := block.MerkleRoot

	err = block.AddData(DataTypeTransaction, "new transaction")
	assert.NoError(t, err)

	assert.NotEqual(t, originalHash, block.Hash)
	assert.NotEqual(t, originalRoot, block.MerkleRoot)
	assert.Len(t, block.Data, 1)
	assert.Equal(t, DataTypeTransaction, block.Data[0].Type)
	assert.Equal(t, "new transaction", block.Data[0].Content)

	// Test adding invalid data type
	err = block.AddData("invalid_type", "content")
	assert.Error(t, err)

	// Test adding nil content
	err = block.AddData(DataTypeTransaction, nil)
	assert.Error(t, err)
}

func TestBlockSerialization(t *testing.T) {
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, []Data{
		{Type: DataTypeTransaction, Content: "test transaction"},
	})
	require.NoError(t, err)

	jsonData, err := json.Marshal(block)
	require.NoError(t, err)

	var deserializedBlock Block
	err = json.Unmarshal(jsonData, &deserializedBlock)
	require.NoError(t, err)

	assert.Equal(t, block.Version, deserializedBlock.Version)
	assert.Equal(t, block.Height, deserializedBlock.Height)
	assert.Equal(t, block.PrevBlockHash, deserializedBlock.PrevBlockHash)
	assert.Equal(t, block.Hash, deserializedBlock.Hash)
	assert.Equal(t, block.MerkleRoot, deserializedBlock.MerkleRoot)
	assert.Equal(t, block.Nonce, deserializedBlock.Nonce)
	assert.Equal(t, block.Data, deserializedBlock.Data)
	assert.Equal(t, block.Timestamp, deserializedBlock.Timestamp)
}

func TestNewBlockEdgeCases(t *testing.T) {
	// Test with negative height
	_, err := NewBlock(testVersion, int64(-1), testPrevBlockHash, []Data{})
	assert.Error(t, err)

	// Test with empty previous hash
	_, err = NewBlock(testVersion, testHeight, "", []Data{})
	assert.Error(t, err)

	// Test with nil data
	block, err := NewBlock(testVersion, testHeight, testPrevBlockHash, nil)
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Empty(t, block.Data)

	// Test with maximum allowed data size (assuming there's a limit)
	maxData := make([]Data, 1000) // Adjust this number based on your actual limit
	for i := range maxData {
		maxData[i] = Data{Type: DataTypeTransaction, Content: "max size test"}
	}
	block, err = NewBlock(testVersion, testHeight, testPrevBlockHash, maxData)
	assert.NoError(t, err)
	assert.NotNil(t, block)
	assert.Len(t, block.Data, 1000)
}

func TestConcurrentAccess(t *testing.T) {
	block, _ := NewBlock(testVersion, testHeight, testPrevBlockHash, []Data{})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := block.AddData(DataTypeTransaction, fmt.Sprintf("tx%d", i))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 100, len(block.Data))

	// Verify all transactions are present and in order
	for i := 0; i < 100; i++ {
		found := false
		for _, data := range block.Data {
			if data.Content == fmt.Sprintf("tx%d", i) {
				found = true
				break
			}
		}
		assert.True(t, found, "Transaction %d not found", i)
	}
}

func BenchmarkNewBlock(b *testing.B) {
	data := []Data{{Type: DataTypeTransaction, Content: "benchmark transaction"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewBlock(testVersion, testHeight, testPrevBlockHash, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

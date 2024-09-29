package core

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockchain(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	assert.NotNil(t, bc)
	assert.Len(t, bc.chain, 1)
	assert.NotNil(t, bc.chain[0])
}

func TestAddBlock(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	initialLength := len(bc.chain)
	latestHash := bc.GetLatestBlock().Hash
	newBlock, err := NewBlock(1, uint64(initialLength), string(latestHash), []Data{
		{Type: DataTypeTransaction, Content: "Test Transaction"},
	})
	require.NoError(t, err)

	err = bc.AddBlock(newBlock)
	assert.NoError(t, err)
	assert.Len(t, bc.chain, initialLength+1)
	assert.Equal(t, newBlock, bc.GetLatestBlock())
}

func TestConcurrentBlockAddition(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	numBlocks := 100
	var wg sync.WaitGroup
	errChan := make(chan error, numBlocks)
	blockChan := make(chan int)

	for i := 1; i <= numBlocks; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			currentHeight := <-blockChan

			latestBlock := bc.GetLatestBlock()

			if int(latestBlock.Height)+1 != currentHeight {
				errChan <- fmt.Errorf("expected to add block at height %d, but latest block height is %d", currentHeight, latestBlock.Height)
				return
			}

			newBlock, err := NewBlock(uint32(1), uint64(currentHeight), string(latestBlock.Hash), []Data{
				{Type: DataTypeTransaction, Content: fmt.Sprintf("Concurrent Transaction %d", currentHeight)},
			})
			if err != nil {
				errChan <- err
				return
			}

			err = bc.AddBlock(newBlock)
			if err != nil {
				errChan <- err
			}

			// Always allow the next goroutine to proceed, even if there was an error
			select {
			case blockChan <- currentHeight + 1:
			default:
				// If the channel is full, we don't want to block
			}
		}(i)
	}

	// Start the process
	go func() {
		blockChan <- 1
	}()

	// Wait for all goroutines to finish or timeout after 10 seconds
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(blockChan) // Close the blockChan to prevent goroutines from blocking
		close(errChan)
	case <-time.After(30 * time.Second):
		close(blockChan) // Close the blockChan to prevent goroutines from blocking
		t.Fatal("Test timed out after 30 seconds")
	}

	for err := range errChan {
		assert.NoError(t, err)
	}

	assert.Equal(t, numBlocks+1, len(bc.chain)) // +1 for genesis block
	assert.NoError(t, bc.ValidateChain())

	// Verify that blocks are in the correct order
	for i := 1; i <= numBlocks; i++ {
		assert.Equal(t, uint64(i), uint64(bc.chain[i].Height))
		assert.Equal(t, fmt.Sprintf("Concurrent Transaction %d", i), bc.chain[i].Data[0].Content)
	}
}

func TestAddInvalidBlock(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	initialLength := len(bc.chain)

	invalidBlock, err := NewBlock(1, 999, string("invalid_hash"), []Data{})
	require.NoError(t, err)

	err = bc.AddBlock(invalidBlock)
	assert.Error(t, err)
	assert.Len(t, bc.chain, initialLength)
}

func TestGetLatestBlock(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	latestBlock := bc.GetLatestBlock()
	assert.NotNil(t, latestBlock)
	assert.Equal(t, bc.chain[len(bc.chain)-1], latestBlock)
}

func TestValidateChain(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	// Add a few valid blocks
	for i := 1; i <= 3; i++ {
		newBlock, err := NewBlock(1, uint64(i), string(bc.GetLatestBlock().Hash), []Data{
			{Type: DataTypeTransaction, Content: "Test Transaction"},
		})
		require.NoError(t, err)
		err = bc.AddBlock(newBlock)
		require.NoError(t, err)
	}

	err = bc.ValidateChain()

	assert.NoError(t, err)

	// Tamper with a block
	bc.chain[2].Data = []Data{{Type: DataTypeTransaction, Content: "Tampered Transaction"}}

	err = bc.ValidateChain()
	assert.Error(t, err)
}

func TestGenesisBlock(t *testing.T) {
	genesisBlock, err := GenesisBlock()
	require.NoError(t, err)
	assert.NotNil(t, genesisBlock)
	assert.Equal(t, BlockHeight(0), genesisBlock.Height)
	assert.Equal(t, BlockHash(""), genesisBlock.PrevBlockHash)
	assert.Len(t, genesisBlock.Data, 1)
	assert.Equal(t, DataTypeTransaction, genesisBlock.Data[0].Type)
	assert.Equal(t, "Genesis Block", genesisBlock.Data[0].Content)
	assert.Equal(t, time.Date(2024, 8, 6, 1, 11, 0, 0, time.UTC), genesisBlock.Timestamp)
	assert.NotEmpty(t, genesisBlock.Hash)
}

func TestAddBlockWithInvalidPrevHash(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	latestBlock := bc.GetLatestBlock()
	invalidBlock, err := NewBlock(1, uint64(latestBlock.Height)+1, "invalid_prev_hash", []Data{
		{Type: DataTypeTransaction, Content: "Invalid Block"},
	})
	require.NoError(t, err)

	err = bc.AddBlock(invalidBlock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid previous block hash")
}

func TestAddBlockWithInvalidHeight(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	latestBlock := bc.GetLatestBlock()
	invalidBlock, err := NewBlock(1, uint64(latestBlock.Height)+2, string(latestBlock.Hash), []Data{
		{Type: DataTypeTransaction, Content: "Invalid Height Block"},
	})
	require.NoError(t, err)

	err = bc.AddBlock(invalidBlock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block height")
}

func TestAddBlockWithFutureTimestamp(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	latestBlock := bc.GetLatestBlock()
	futureBlock, err := NewBlock(1, uint64(latestBlock.Height)+1, string(latestBlock.Hash), []Data{
		{Type: DataTypeTransaction, Content: "Future Block"},
	})
	require.NoError(t, err)
	futureBlock.Timestamp = time.Now().Add(time.Hour) // Set timestamp to 1 hour in the future

	err = bc.AddBlock(futureBlock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block timestamp")
}

func TestConcurrentBlockAdditionWithErrors(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	numBlocks := 100
	var wg sync.WaitGroup
	errChan := make(chan error, numBlocks)
	blockChan := make(chan int, 1)

	for i := 1; i <= numBlocks; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			currentHeight := <-blockChan

			latestBlock := bc.GetLatestBlock()

			var newBlock *Block
			var err error

			if i%10 == 0 { // Intentionally create an invalid block every 10th iteration
				newBlock, err = NewBlock(uint32(1), uint64(currentHeight+1), string(latestBlock.Hash), []Data{
					{Type: DataTypeTransaction, Content: fmt.Sprintf("Invalid Concurrent Transaction %d", currentHeight)},
				})
			} else {
				newBlock, err = NewBlock(uint32(1), uint64(currentHeight), string(latestBlock.Hash), []Data{
					{Type: DataTypeTransaction, Content: fmt.Sprintf("Concurrent Transaction %d", currentHeight)},
				})
			}

			if err != nil {
				errChan <- err
				blockChan <- currentHeight // Allow next goroutine to proceed
				return
			}

			err = bc.AddBlock(newBlock)
			if err != nil {
				errChan <- err
				blockChan <- currentHeight // Allow next goroutine to proceed without incrementing height
			} else {
				blockChan <- currentHeight + 1 // Increment height only for successful additions
			}
		}(i)
	}

	blockChan <- 1 // Start the process

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(blockChan)
		close(errChan)
	case <-time.After(30 * time.Second):
		close(blockChan)
		t.Fatal("Test timed out after 30 seconds")
	}

	errorCount := 0
	for err := range errChan {
		t.Logf("Error encountered: %v", err)
		errorCount++
	}

	expectedErrorCount := numBlocks / 10
	assert.Equal(t, expectedErrorCount, errorCount, "Expected 10% of blocks to have errors")
	expectedBlockCount := numBlocks - errorCount + 1 // +1 for genesis block
	assert.Equal(t, expectedBlockCount, len(bc.chain), "Unexpected number of blocks in chain")
	assert.NoError(t, bc.ValidateChain())
}

func TestAddLargeNumberOfBlocks(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)
	numBlocks := 10000

	for i := 1; i <= numBlocks; i++ {
		latestBlock := bc.GetLatestBlock()
		newBlock, err := NewBlock(1, uint64(i), string(latestBlock.Hash), []Data{
			{Type: DataTypeTransaction, Content: fmt.Sprintf("Transaction %d", i)},
		})
		require.NoError(t, err)

		err = bc.AddBlock(newBlock)
		assert.NoError(t, err)
	}

	assert.Equal(t, numBlocks+1, len(bc.chain)) // +1 for genesis block
	assert.NoError(t, bc.ValidateChain())
}

func TestValidateChainWithTamperedIntermediateBlock(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)

	// Add 10 valid blocks
	for i := 1; i <= 10; i++ {
		newBlock, err := NewBlock(1, uint64(i), string(bc.GetLatestBlock().Hash), []Data{
			{Type: DataTypeTransaction, Content: fmt.Sprintf("Transaction %d", i)},
		})
		require.NoError(t, err)
		err = bc.AddBlock(newBlock)
		require.NoError(t, err)
	}

	// Tamper with an intermediate block (e.g., the 5th block)
	bc.chain[5].Data = []Data{{Type: DataTypeTransaction, Content: "Tampered Transaction"}}

	err = bc.ValidateChain()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block at height 5")
}

func TestGetBlockByHeight(t *testing.T) {
	bc, err := NewBlockchain()
	require.NoError(t, err)

	// Add 5 blocks
	for i := 1; i <= 5; i++ {
		newBlock, err := NewBlock(1, uint64(i), string(bc.GetLatestBlock().Hash), []Data{
			{Type: DataTypeTransaction, Content: fmt.Sprintf("Transaction %d", i)},
		})
		require.NoError(t, err)
		err = bc.AddBlock(newBlock)
		require.NoError(t, err)
	}

	// Test getting existing blocks
	for i := 0; i <= 5; i++ {
		block, err := bc.GetBlockByHeight(uint64(i))
		assert.NoError(t, err)
		assert.Equal(t, uint64(i), uint64(block.Height))
	}

	// Test getting non-existent block
	_, err = bc.GetBlockByHeight(6)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block not found")
}

// Add this method to the Blockchain struct in blockchain.go
func (bc *Blockchain) GetBlockByHeight(height uint64) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if int(height) >= len(bc.chain) {
		return nil, fmt.Errorf("block not found")
	}

	return bc.chain[height], nil
}

package core

import (
	"testing"
	"time"
)

func FuzzBlockchainAddBlock(f *testing.F) {
	// Seed corpus
	f.Add(uint32(1), uint64(1), "prevHash", "Transaction 1")

	f.Fuzz(func(t *testing.T, version uint32, height uint64, prevHash string, transactionContent string) {
		bc, err := NewBlockchain()
		if err != nil {
			t.Skip("Failed to create blockchain")
		}

		newBlock, err := NewBlock(version, height, prevHash, []Data{
			{Type: DataTypeTransaction, Content: transactionContent},
		})
		if err != nil {
			return // Invalid input, not a bug
		}

		err = bc.AddBlock(newBlock)
		if err != nil {
			// Error is expected for invalid blocks, not a bug
			return
		}

		// If block was added successfully, validate the chain
		if err := bc.ValidateChain(); err != nil {
			t.Errorf("Chain validation failed after adding block: %v", err)
		}
	})
}

func FuzzBlockchainValidateBlock(f *testing.F) {
	// Seed corpus
	f.Add(uint32(1), uint64(1), "prevHash", "Transaction 1", int64(1630000000))

	f.Fuzz(func(t *testing.T, version uint32, height uint64, prevHash string, transactionContent string, timestamp int64) {
		bc, err := NewBlockchain()
		if err != nil {
			t.Skip("Failed to create blockchain")
		}

		newBlock, err := NewBlock(version, height, prevHash, []Data{
			{Type: DataTypeTransaction, Content: transactionContent},
		})
		if err != nil {
			return // Invalid input, not a bug
		}

		newBlock.Timestamp = time.Unix(timestamp, 0)

		err = bc.validateBlock(newBlock, bc.GetLatestBlock())
		// We don't assert on the error here because we're interested in panics or other unexpected behavior
	})
}

func FuzzBlockchainCalculateHash(f *testing.F) {
	f.Add(uint32(1), uint64(1), "prevHash", "Transaction 1", int64(1630000000))

	f.Fuzz(func(t *testing.T, version uint32, height uint64, prevHash string, transactionContent string, timestamp int64) {
		block, err := NewBlock(version, height, prevHash, []Data{
			{Type: DataTypeTransaction, Content: transactionContent},
		})
		if err != nil {
			return // Invalid input, not a bug
		}

		block.Timestamp = time.Unix(timestamp, 0)

		_, err = block.CalculateHash()
		if err != nil {
			t.Errorf("Failed to calculate hash: %v", err)
		}
	})
}

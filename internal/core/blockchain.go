package core

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/4ritam/kaizen-blockchain/internal/crypto"
)

type Blockchain struct {
	chain []*Block
	mu    sync.RWMutex
}

func NewBlockchain() (*Blockchain, error) {
	genesisBlock, err := GenesisBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block: %v", err)
	}

	return &Blockchain{
		chain: []*Block{genesisBlock},
	}, nil
}

func (bc *Blockchain) AddBlock(block *Block) error {
	// log.Printf("Starting to add block at height %d", block.Height)
	bc.mu.Lock()
	defer bc.mu.Unlock()

	prevBlock := bc.chain[len(bc.chain)-1]
	if err := bc.validateBlockWithoutLock(block, prevBlock); err != nil {
		// log.Printf("Failed to validate block at height %d: %v", block.Height, err)
		return err
	}

	bc.chain = append(bc.chain, block)
	// log.Printf("Successfully added block at height %d", block.Height)
	return nil
}

func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) validateBlock(block *Block, prevBlock *Block) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.validateBlockWithoutLock(block, prevBlock)
}

func (bc *Blockchain) validateBlockWithoutLock(block *Block, prevBlock *Block) error {
	if block.Height == 0 {
		// Special case for genesis block
		if block.PrevBlockHash != "" {
			return errors.New("genesis block must have empty previous block hash")
		}
		// Add any other genesis block-specific validations here
		return nil
	}

	// Check if the block's previous hash matches the hash of the previous block
	if block.PrevBlockHash != prevBlock.Hash {
		return errors.New("invalid previous block hash")
	}

	// Check if the block height is correct
	if block.Height != prevBlock.Height+1 {
		return errors.New("invalid block height")
	}

	// Check if the block timestamp is valid (not in the future and after the previous block)
	if block.Timestamp.After(time.Now()) || block.Timestamp.Before(prevBlock.Timestamp) {
		return errors.New("invalid block timestamp")
	}

	// Validate the block's hash
	if !crypto.ValidateHash(string(block.Hash)) {
		return errors.New("invalid block hash format")
	}

	// Recalculate the block hash and compare
	calculatedHash, err := block.CalculateHash()
	if err != nil {
		return err
	}
	if calculatedHash != block.Hash {
		return errors.New("block hash mismatch")
	}

	// Validate the Merkle root
	merkleRoot, err := block.calculateMerkleRoot()
	if err != nil {
		return err
	}
	if merkleRoot != block.MerkleRoot {
		return errors.New("invalid Merkle root")
	}

	// Additional validations can be added here (e.g., checking the nonce for PoW)

	return nil
}

func (bc *Blockchain) ValidateChain() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	log.Printf("Validating chain with %d blocks", len(bc.chain))
	for i := 1; i < len(bc.chain); i++ {
		if err := bc.validateBlockWithoutLock(bc.chain[i], bc.chain[i-1]); err != nil {
			return fmt.Errorf("invalid block at height %d: %v", bc.chain[i].Height, err)
		}
	}
	return nil
}

func GenesisBlock() (*Block, error) {
	genesisData := []Data{
		{
			Type:    DataTypeTransaction,
			Content: "Genesis Block",
		},
	}

	block, err := NewBlock(
		uint32(1), // Version
		uint64(0), // Height
		"",        // PrevBlockHash (empty for genesis block)
		genesisData,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block: %v", err)
	}

	// Set a fixed timestamp for the genesis block
	block.Timestamp = time.Date(2024, 8, 6, 1, 11, 0, 0, time.UTC) // Use a fixed date for reproducibility
	block.Hash, err = block.CalculateHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate genesis block hash: %v", err)
	}
	return block, nil
}

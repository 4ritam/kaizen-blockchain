package miner

import (
	"math/big"
	"runtime"
	"time"

	"github.com/4ritam/kaizen-blockchain/internal/core"
)

// Miner represents a miner that can mine blocks
type Miner struct {
	difficulty *big.Int
}

// Mine performs the mining operation on a given block
func (m *Miner) Mine(block *core.Block) error {
	var (
		target     = new(big.Int).Set(m.difficulty)
		numCPU     = runtime.NumCPU()
		stopChan   = make(chan struct{})
		resultChan = make(chan uint64)
	)

	for i := 0; i < numCPU; i++ {
		go m.mineRoutine(block, uint64(i), uint64(numCPU), target, stopChan, resultChan)
	}

	// Wait for a valid nonce or timeout
	select {
	case nonce := <-resultChan:
		close(stopChan)
		block.Nonce = core.BlockNonce(nonce)
		hash, err := block.CalculateHash()
		if err != nil {
			return err
		}
		block.Hash = hash
	case <-time.After(10 * time.Minute):
		close(stopChan)
	}
	return nil
}

func (m *Miner) mineRoutine(block *core.Block, start, step uint64, target *big.Int, stop <-chan struct{}, result chan<- uint64) error {
	hashInt := new(big.Int)
	nonce := start

	for {
		select {
		case <-stop:
			return nil
		default:
			block.Nonce = core.BlockNonce(nonce)
			hash, err := block.CalculateHash()
			if err != nil {
				return err
			}
			hashInt.SetString(string(hash), 16)

			if hashInt.Cmp(target) == -1 {
				result <- nonce
				return nil
			}

			nonce += step
		}
	}
}

// SetDifficulty sets the mining difficulty
func (m *Miner) SetDifficulty(difficulty uint) {
	m.difficulty = big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256-int64(difficulty)), nil)
}

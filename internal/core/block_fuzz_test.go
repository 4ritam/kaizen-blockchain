package core

import (
	"testing"
	"time"
)

func FuzzBlockNewBlock(f *testing.F) {
	f.Add(uint32(1), uint64(1), "prevHash", "Transaction 1")
	f.Fuzz(func(t *testing.T, version uint32, height uint64, prevHash string, transactionContent string) {
		_, err := NewBlock(version, height, prevHash, []Data{
			{Type: DataTypeTransaction, Content: transactionContent},
		})
		if err != nil {
			return // Invalid input, not a bug
		}
	})
}

func FuzzBlockCalculateHash(f *testing.F) {
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

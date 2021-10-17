package abft

import (
	"github.com/nexgencoin/nexgen-base/inter/idx"
	"github.com/nexgencoin/nexgen-base/inter/pos"
	"github.com/nexgencoin/nexgen-base/kvdb"
	"github.com/nexgencoin/nexgen-base/kvdb/memorydb"
	"github.com/nexgencoin/nexgen-base/nexgenbft"
	"github.com/nexgencoin/nexgen-base/utils/adapters"
	"github.com/nexgencoin/nexgen-base/vecfc"
)

type applyBlockFn func(block *nexgenbft.Block) *pos.Validators

// TestNexgenBFT extends NexgenBFT for tests.
type TestNexgenBFT struct {
	*IndexedNexgenBFT

	blocks map[idx.Block]*nexgenbft.Block

	applyBlock applyBlockFn
}

// FakeNexgenBFT creates empty abft with mem store and equal weights of nodes in genesis.
func FakeNexgenBFT(nodes []idx.ValidatorID, weights []pos.Weight, mods ...memorydb.Mod) (*TestNexgenBFT, *Store, *EventStore) {
	validators := make(pos.ValidatorsBuilder, len(nodes))
	for i, v := range nodes {
		if weights == nil {
			validators[v] = 1
		} else {
			validators[v] = weights[i]
		}
	}

	openEDB := func(epoch idx.Epoch) kvdb.DropableStore {
		return memorydb.New()
	}
	crit := func(err error) {
		panic(err)
	}
	store := NewStore(memorydb.New(), openEDB, crit, LiteStoreConfig())

	err := store.ApplyGenesis(&Genesis{
		Validators: validators.Build(),
		Epoch:      FirstEpoch,
	})
	if err != nil {
		panic(err)
	}

	input := NewEventStore()

	config := LiteConfig()
	lch := NewIndexedNexgenBFT(store, input, &adapters.VectorToDagIndexer{vecfc.NewIndex(crit, vecfc.LiteConfig())}, crit, config)

	extended := &TestNexgenBFT{
		IndexedNexgenBFT: lch,
		blocks:          map[idx.Block]*nexgenbft.Block{},
	}

	blockIdx := idx.Block(0)

	err = extended.Bootstrap(nexgenbft.ConsensusCallbacks{
		BeginBlock: func(block *nexgenbft.Block) nexgenbft.BlockCallbacks {
			blockIdx++
			return nexgenbft.BlockCallbacks{
				EndBlock: func() (sealEpoch *pos.Validators) {
					// track blocks
					extended.blocks[blockIdx] = block
					if extended.applyBlock != nil {
						return extended.applyBlock(block)
					}
					return nil
				},
			}
		},
	})
	if err != nil {
		panic(err)
	}

	return extended, store, input
}

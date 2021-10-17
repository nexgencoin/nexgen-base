package abft

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/nexgencoin/nexgen-base/abft/dagidx"
	"github.com/nexgencoin/nexgen-base/hash"
	"github.com/nexgencoin/nexgen-base/inter/dag"
	"github.com/nexgencoin/nexgen-base/inter/idx"
	"github.com/nexgencoin/nexgen-base/inter/pos"
	"github.com/nexgencoin/nexgen-base/kvdb"
	"github.com/nexgencoin/nexgen-base/nexgenbft"
)

var _ nexgenbft.Consensus = (*IndexedNexgenBFT)(nil)

// IndexedNexgenBFT performs events ordering and detects cheaters
// It's a wrapper around Orderer, which adds features which might potentially be application-specific:
// confirmed events traversal, DAG index updates and cheaters detection.
// Use this structure if need a general-purpose consensus. Instead, use lower-level abft.Orderer.
type IndexedNexgenBFT struct {
	*NexgenBFT
	dagIndexer    DagIndexer
	uniqueDirtyID uniqueID
}

type DagIndexer interface {
	dagidx.VectorClock
	dagidx.ForklessCause

	Add(dag.Event) error
	Flush()
	DropNotFlushed()

	Reset(validators *pos.Validators, db kvdb.Store, getEvent func(hash.Event) dag.Event)
}

// New creates IndexedNexgenBFT instance.
func NewIndexedNexgenBFT(store *Store, input EventSource, dagIndexer DagIndexer, crit func(error), config Config) *IndexedNexgenBFT {
	p := &IndexedNexgenBFT{
		NexgenBFT:      NewNexgenBFT(store, input, dagIndexer, crit, config),
		dagIndexer:    dagIndexer,
		uniqueDirtyID: uniqueID{new(big.Int)},
	}

	return p
}

// Build fills consensus-related fields: Frame, IsRoot
// returns error if event should be dropped
func (p *IndexedNexgenBFT) Build(e dag.MutableEvent) error {
	e.SetID(p.uniqueDirtyID.sample())

	defer p.dagIndexer.DropNotFlushed()
	err := p.dagIndexer.Add(e)
	if err != nil {
		return err
	}

	return p.NexgenBFT.Build(e)
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *IndexedNexgenBFT) Process(e dag.Event) (err error) {
	defer p.dagIndexer.DropNotFlushed()
	err = p.dagIndexer.Add(e)
	if err != nil {
		return err
	}

	err = p.NexgenBFT.Process(e)
	if err != nil {
		return err
	}
	p.dagIndexer.Flush()
	return nil
}

func (p *IndexedNexgenBFT) Bootstrap(callback nexgenbft.ConsensusCallbacks) error {
	base := p.NexgenBFT.OrdererCallbacks()
	ordererCallbacks := OrdererCallbacks{
		ApplyAtropos: base.ApplyAtropos,
		EpochDBLoaded: func(epoch idx.Epoch) {
			if base.EpochDBLoaded != nil {
				base.EpochDBLoaded(epoch)
			}
			p.dagIndexer.Reset(p.store.GetValidators(), p.store.epochTable.VectorIndex, p.input.GetEvent)
		},
	}
	return p.NexgenBFT.BootstrapWithOrderer(callback, ordererCallbacks)
}

type uniqueID struct {
	counter *big.Int
}

func (u *uniqueID) sample() [24]byte {
	u.counter = u.counter.Add(u.counter, common.Big1)
	var id [24]byte
	copy(id[:], u.counter.Bytes())
	return id
}

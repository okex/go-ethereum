package snapshot

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"time"
)

// Retriever custom
type Retriever interface {
	RetrieveStateRoot(bz []byte) common.Hash
}

// NewCustom attempts to load an already existing snapshot from a persistent key-value
// store (with a number of memory layers from a journal), ensuring that the head
// of the snapshot matches the expected one.
//
// If the snapshot is missing or the disk layer is broken, the snapshot will be
// reconstructed using both the existing data and the state trie.
// The repair happens on a background thread.
//
// If the memory layers in the journal do not match the disk layer (e.g. there is
// a gap) or the journal is missing, there are two repair cases:
//
// - if the 'recovery' parameter is true, all memory diff-layers will be discarded.
//   This case happens when the snapshot is 'ahead' of the state trie.
// - otherwise, the entire snapshot is considered invalid and will be recreated on
//   a background thread.
//
// NewCustom use retriever to decode Account
func NewCustom(diskdb ethdb.KeyValueStore, triedb *trie.Database, cache int, root common.Hash, async bool, rebuild bool, recovery bool, retriever Retriever) (*Tree, error) {
	// Create a new, empty snapshot tree
	snap := &Tree{
		diskdb:    diskdb,
		triedb:    triedb,
		cache:     cache,
		layers:    make(map[common.Hash]snapshot),
		Retriever: retriever,
	}
	if !async {
		defer snap.waitBuild()
	}
	// Attempt to load a previously persisted snapshot and rebuild one if failed
	head, disabled, err := loadSnapshot(diskdb, triedb, cache, root, recovery)
	if disabled {
		log.Warn("Snapshot maintenance disabled (syncing)")
		return snap, nil
	}
	if err != nil {
		if rebuild {
			log.Warn("Failed to load snapshot, regenerating", "err", err)
			snap.Rebuild(root)
			return snap, nil
		}
		return nil, err // Bail out the error, don't rebuild automatically.
	}
	// Existing snapshot loaded, seed all the layers
	for head != nil {
		snap.layers[head.Root()] = head
		head = head.Parent()
	}
	return snap, nil
}

// generateSnapshotCustom regenerates a brand new snapshot based on an existing state
// database and head block asynchronously. The snapshot is returned immediately
// and generation is continued in the background until done.
//
// generateSnapshotCustom is transformed by the retriever to decode the account
func generateSnapshotCustom(diskdb ethdb.KeyValueStore, triedb *trie.Database, cache int, root common.Hash, retriever Retriever) *diskLayer {
	// Create a new disk layer with an initialized state marker at zero
	var (
		stats     = &generatorStats{start: time.Now()}
		batch     = diskdb.NewBatch()
		genMarker = []byte{} // Initialized but empty!
	)
	rawdb.WriteSnapshotRoot(batch, root)
	journalProgress(batch, genMarker, stats)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write initialized state marker", "err", err)
	}
	base := &diskLayer{
		diskdb:     diskdb,
		triedb:     triedb,
		root:       root,
		cache:      fastcache.New(cache * 1024 * 1024),
		genMarker:  genMarker,
		genPending: make(chan struct{}),
		genAbort:   make(chan chan *generatorStats),
		Retriever:  retriever,
	}
	go base.generate(stats)
	log.Debug("Start snapshot generation", "root", root)
	return base
}

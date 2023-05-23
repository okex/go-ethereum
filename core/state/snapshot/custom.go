package snapshot

import (
	"bytes"
	"fmt"
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"runtime"
	"sync"
	"time"
)

var (
	zeroRoot = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000")
)

// Retriever custom
type Retriever interface {
	RetrieveStateRoot(bz []byte) common.Hash
	GetStateRootAndCodeHash(bz []byte) (common.Hash, []byte)
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
//   - if the 'recovery' parameter is true, all memory diff-layers will be discarded.
//     This case happens when the snapshot is 'ahead' of the state trie.
//   - otherwise, the entire snapshot is considered invalid and will be recreated on
//     a background thread.
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

// generateTrieRootCustom generates the trie hash based on the snapshot iterator.
// It can be used for generating account trie, storage trie or even the
// whole state which connects the accounts and the corresponding storages.
func generateTrieRootCustom(db ethdb.KeyValueWriter, it Iterator, account common.Hash, generatorFn trieGeneratorFn, leafCallback leafCallbackFn, stats *generateStats, report bool, retriever Retriever) (common.Hash, error) {
	var (
		in      = make(chan trieKV)         // chan to pass leaves
		out     = make(chan common.Hash, 1) // chan to collect result
		stoplog = make(chan bool, 1)        // 1-size buffer, works when logging is not enabled
		wg      sync.WaitGroup
	)
	// Spin up a go-routine for trie hash re-generation
	wg.Add(1)
	go func() {
		defer wg.Done()
		generatorFn(db, account, in, out)
	}()
	// Spin up a go-routine for progress logging
	if report && stats != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runReport(stats, stoplog)
		}()
	}
	// Create a semaphore to assign tasks and collect results through. We'll pre-
	// fill it with nils, thus using the same channel for both limiting concurrent
	// processing and gathering results.
	threads := runtime.NumCPU()
	results := make(chan error, threads)
	for i := 0; i < threads; i++ {
		results <- nil // fill the semaphore
	}
	// stop is a helper function to shutdown the background threads
	// and return the re-generated trie hash.
	stop := func(fail error) (common.Hash, error) {
		close(in)
		result := <-out
		for i := 0; i < threads; i++ {
			if err := <-results; err != nil && fail == nil {
				fail = err
			}
		}
		stoplog <- fail == nil

		wg.Wait()
		return result, fail
	}
	var (
		logged    = time.Now()
		processed = uint64(0)
		leaf      trieKV
	)
	// Start to feed leaves
	for it.Next() {
		if account == (common.Hash{}) {
			var (
				// err      error
				fullData []byte
			)
			if leafCallback == nil {
				//			fullData, err = FullAccountRLP(it.(AccountIterator).Account())
				//			if err != nil {
				//				return stop(err)
				//			}
				fullData = it.(AccountIterator).Account()
			} else {
				// Wait until the semaphore allows us to continue, aborting if
				// a sub-task failed
				if err := <-results; err != nil {
					results <- nil // stop will drain the results, add a noop back for this error we just consumed
					return stop(err)
				}
				// Fetch the next account and process it concurrently
				account, err := FullAccountCustom(it.(AccountIterator).Account(), retriever)
				if err != nil {
					return stop(err)
				}
				go func(hash common.Hash) {
					subroot, err := leafCallback(db, hash, common.BytesToHash(account.CodeHash), stats)
					if err != nil {
						results <- err
						return
					}
					if !bytes.Equal(account.Root, subroot.Bytes()) {
						results <- fmt.Errorf("invalid subroot(path %x), want %x, have %x", hash, account.Root, subroot)
						return
					}
					results <- nil
				}(it.Hash())
				fullData = it.(AccountIterator).Account()
				//fullData, err = rlp.EncodeToBytes(account)
				//if err != nil {
				//	return stop(err)
				//}
			}
			leaf = trieKV{it.Hash(), common.CopyBytes(fullData)}
		} else {
			leaf = trieKV{it.Hash(), common.CopyBytes(it.(StorageIterator).Slot())}
		}
		in <- leaf

		// Accumulate the generation statistic if it's required.
		processed++
		if time.Since(logged) > 3*time.Second && stats != nil {
			if account == (common.Hash{}) {
				stats.progressAccounts(it.Hash(), processed)
			} else {
				stats.progressContract(account, it.Hash(), processed)
			}
			logged, processed = time.Now(), 0
		}
	}
	// Commit the last part statistic.
	if processed > 0 && stats != nil {
		if account == (common.Hash{}) {
			stats.finishAccounts(processed)
		} else {
			stats.finishContract(account, processed)
		}
	}
	return stop(nil)
}

// FullAccountCustom decodes the data on the 'slim RLP' format and return
// the consensus format account.
func FullAccountCustom(data []byte, retriever Retriever) (Account, error) {
	var account Account

	stateRoot, codeHash := retriever.GetStateRootAndCodeHash(data)
	account.Root = stateRoot[:]
	account.CodeHash = codeHash[:]

	if len(account.Root) == 0 || bytes.Compare(zeroRoot[:], account.Root) == 0 {
		account.Root = emptyRoot[:]
	}
	if len(account.CodeHash) == 0 || bytes.Compare(zeroRoot[:], account.CodeHash) == 0 {
		account.CodeHash = emptyCode[:]
	}
	return account, nil
}

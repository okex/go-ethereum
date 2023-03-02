package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"sync"
	"time"
)

const (
	batchChanLen = 100
)

type keyvalue struct {
	key    []byte
	value  []byte
	delete bool
}

type ACProcessor struct {
	kvdatas chan []*keyvalue
	cache   *Cache
	diskdb  ethdb.KeyValueStore
}

func NewACProcessor(db ethdb.KeyValueStore) *ACProcessor {
	ac := &ACProcessor{
		kvdatas: make(chan []*keyvalue, batchChanLen),
		cache:   NewCache(),
		diskdb:  db,
	}
	go ac.ACCommit()
	return ac
}

func (ac *ACProcessor) Close() {
	close(ac.kvdatas)
}

func (ac *ACProcessor) ACCommit() {
	batch := ac.diskdb.NewBatch()
	for kvs := range ac.kvdatas {
		for _, kv := range kvs {
			if !kv.delete {
				batch.Delete(kv.key)
			} else {
				batch.Put(kv.key, kv.value)
			}
		}
		batch.Write()
		ac.cache.Clear(kvs)
	}
}

type Cache struct {
	dirties   map[common.Hash]*cachedNode
	preimages map[common.Hash][]byte
	lock      sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		dirties:   make(map[common.Hash]*cachedNode),
		preimages: make(map[common.Hash][]byte),
	}
}

func (c *Cache) SetDirty(key common.Hash, node *cachedNode) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.dirties[key] = node
}

func (c *Cache) GetDirty(key common.Hash) (*cachedNode, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if n, ok := c.dirties[key]; ok {
		return n, true
	}
	return nil, false
}

func (c *Cache) SetPreimages(images map[common.Hash][]byte) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for k, v := range images {
		c.preimages[k] = v
	}
}

func (c *Cache) GetPreimage(key common.Hash) ([]byte, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if v, ok := c.preimages[key]; ok {
		return v, true
	}
	return nil, false
}

func (c *Cache) Clear(kvs []*keyvalue) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, kv := range kvs {
		hash := common.BytesToHash(kv.key)
		if _, ok := c.dirties[hash]; ok {
			delete(c.dirties, hash)
		} else {
			delete(c.preimages, hash)
		}
	}
}

type BatchEx struct {
	writes []*keyvalue
	size   int
	data   chan []*keyvalue
}

func NewBatchEx(kvchan chan []*keyvalue) *BatchEx {
	return &BatchEx{
		data: kvchan,
	}
}

// Put inserts the given value into the key-value data store.
func (b *BatchEx) Put(key []byte, value []byte) error {
	b.writes = append(b.writes, &keyvalue{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(key) + len(value)
	return nil
}

// Delete removes the key from the key-value data store.
func (b *BatchEx) Delete(key []byte) error {
	b.writes = append(b.writes, &keyvalue{common.CopyBytes(key), nil, true})
	b.size += len(key)
	return nil
}

func (b *BatchEx) ValueSize() int {
	return b.size
}

func (b *BatchEx) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

func (b *BatchEx) Write() error {
	b.data <- b.writes
	return nil
}

func (b *BatchEx) Replay(w ethdb.KeyValueWriter) error {
	for _, keyvalue := range b.writes {
		if keyvalue.delete {
			if err := w.Delete(keyvalue.key); err != nil {
				return err
			}
			continue
		}
		if err := w.Put(keyvalue.key, keyvalue.value); err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) AsyncCommitV8(node common.Hash, report bool, callback func(common.Hash)) error {
	// Create a database batch to flush persistent data out. It is important that
	// outside code doesn't see an inconsistent state (referenced data removed from
	// memory cache during commit but not yet in persistent storage). This is ensured
	// by only uncaching existing data when the database write finalizes.
	start := time.Now()
	batch := NewBatchEx(db.acProcessor.kvdatas)

	// Move all of the accumulated preimages into a write batch
	if db.preimages != nil {
		rawdb.WritePreimages(batch, db.preimages)
		// Since we're going to replay trie node writes into the clean cache, flush out
		// any batched pre-images before continuing.
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
		db.acProcessor.cache.SetPreimages(db.preimages)
	}
	// Move the trie itself into the batch, flushing if enough data is accumulated
	nodes, storage := len(db.dirties), db.dirtiesSize

	uncacher := &cleaner{db}
	if err := db.asyncCommit(node, batch, uncacher, callback); err != nil {
		log.Error("Failed to commit trie from trie database", "err", err)
		return err
	}
	// Trie mostly committed to disk, flush any batch leftovers
	if err := batch.Write(); err != nil {
		log.Error("Failed to write trie to disk", "err", err)
		return err
	}
	// Uncache any leftovers in the last batch
	db.lock.Lock()
	defer db.lock.Unlock()

	batch.Replay(uncacher)
	batch.Reset()

	// Reset the storage counters and bumped metrics
	if db.preimages != nil {
		db.preimages, db.preimagesSize = make(map[common.Hash][]byte), 0
	}
	memcacheCommitTimeTimer.Update(time.Since(start))
	memcacheCommitSizeMeter.Mark(int64(storage - db.dirtiesSize))
	memcacheCommitNodesMeter.Mark(int64(nodes - len(db.dirties)))

	logger := log.Info
	if !report {
		logger = log.Debug
	}
	logger("Persisted trie from memory database", "nodes", nodes-len(db.dirties)+int(db.flushnodes), "size", storage-db.dirtiesSize+db.flushsize, "time", time.Since(start)+db.flushtime,
		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.dirties), "livesize", db.dirtiesSize)

	// Reset the garbage collection statistics
	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
	db.flushnodes, db.flushsize, db.flushtime = 0, 0, 0

	return nil
}

// commit is the private locked version of Commit.
func (db *Database) asyncCommit(hash common.Hash, batch ethdb.Batch, uncacher *cleaner, callback func(common.Hash)) error {
	// If the node does not exist, it's a previously committed node
	node, ok := db.dirties[hash]
	if !ok {
		return nil
	}
	var err error
	node.forChilds(func(child common.Hash) {
		if err == nil {
			err = db.asyncCommit(child, batch, uncacher, callback)
		}
	})
	if err != nil {
		return err
	}
	// If we've reached an optimal batch size, commit and start over
	db.acProcessor.cache.SetDirty(hash, node)
	rawdb.WriteTrieNode(batch, hash, node.rlp())
	if callback != nil {
		callback(hash)
	}
	if batch.ValueSize() >= ethdb.IdealBatchSize {
		if err := batch.Write(); err != nil {
			return err
		}
		db.lock.Lock()
		batch.Replay(uncacher)
		batch.Reset()
		db.lock.Unlock()
	}
	return nil
}

// AsyncCommit iterates over all the children of a particular node, writes them out
// to disk, forcefully tearing down all references in both directions. As a side
// effect, all pre-images accumulated up to this point are also written.
//
// Note, this method is a non-synchronized mutator. It is unsafe to call this
// concurrently with other mutators.
//func (db *Database) AsyncCommitV25(node common.Hash, report bool, callback func(common.Hash)) error {
//	// Create a database batch to flush persistent data out. It is important that
//	// outside code doesn't see an inconsistent state (referenced data removed from
//	// memory cache during commit but not yet in persistent storage). This is ensured
//	// by only uncaching existing data when the database write finalizes.
//	start := time.Now()
//	batch := NewBatchEx(db.acProcessor.kvdatas)
//
//	// Move all of the accumulated preimages into a write batch
//	if db.preimages != nil {
//		db.preimages.asyncCommit(true, db)
//	}
//	// Move the trie itself into the batch, flushing if enough data is accumulated
//	nodes, storage := len(db.dirties), db.dirtiesSize
//
//	uncacher := &cleaner{db}
//	if err := db.asyncCommit(node, batch, uncacher, callback); err != nil {
//		log.Error("Failed to commit trie from trie database", "err", err)
//		return err
//	}
//	// Trie mostly committed to disk, flush any batch leftovers
//	if err := batch.Write(); err != nil {
//		log.Error("Failed to write trie to disk", "err", err)
//		return err
//	}
//	// Uncache any leftovers in the last batch
//	db.lock.Lock()
//	defer db.lock.Unlock()
//
//	batch.Replay(uncacher)
//	batch.Reset()
//
//	// Reset the storage counters and bumped metrics
//	memcacheCommitTimeTimer.Update(time.Since(start))
//	memcacheCommitSizeMeter.Mark(int64(storage - db.dirtiesSize))
//	memcacheCommitNodesMeter.Mark(int64(nodes - len(db.dirties)))
//
//	logger := log.Info
//	if !report {
//		logger = log.Debug
//	}
//	logger("Persisted trie from memory database", "nodes", nodes-len(db.dirties)+int(db.flushnodes), "size", storage-db.dirtiesSize+db.flushsize, "time", time.Since(start)+db.flushtime,
//		"gcnodes", db.gcnodes, "gcsize", db.gcsize, "gctime", db.gctime, "livenodes", len(db.dirties), "livesize", db.dirtiesSize)
//
//	// Reset the garbage collection statistics
//	db.gcnodes, db.gcsize, db.gctime = 0, 0, 0
//	db.flushnodes, db.flushsize, db.flushtime = 0, 0, 0
//
//	return nil
//}

//// commit flushes the cached preimages into the disk.
//func (store *preimageStore) asyncCommit(force bool, db *Database) error {
//	store.lock.Lock()
//	defer store.lock.Unlock()
//
//	if store.preimagesSize <= 4*1024*1024 && !force {
//		return nil
//	}
//	batch := NewBatchEx(db.acProcessor.kvdatas)
//	rawdb.WritePreimages(batch, store.preimages)
//	if err := batch.Write(); err != nil {
//		return err
//	}
//	db.acProcessor.cache.SetPreimages(store.preimages)
//	store.preimages, store.preimagesSize = make(map[common.Hash][]byte), 0
//	return nil
//}

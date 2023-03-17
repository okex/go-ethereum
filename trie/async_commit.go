package trie

import (
	"bytes"
	"fmt"
	xxhash "github.com/cespare/xxhash/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"sync"
)

const (
	batchChanLen = 100000
	CacheCount   = 100
)

type keyvalue struct {
	key    []byte
	value  []byte
	delete bool
}

type ACProcessor struct {
	kvdatas    chan []*keyvalue
	clearChans chan []*keyvalue
	cache      *CacheList
	diskdb     ethdb.KeyValueStore
}

func NewACProcessor(db ethdb.KeyValueStore) *ACProcessor {
	ac := &ACProcessor{
		kvdatas:    make(chan []*keyvalue, batchChanLen),
		clearChans: make(chan []*keyvalue, batchChanLen),
		cache:      NewCacheList(),
		diskdb:     db,
	}
	go ac.ACCommit()
	go ac.Clear()
	return ac
}

func (ac *ACProcessor) Close() {
	close(ac.kvdatas)
}

func (ac *ACProcessor) ACCommit() {
	for kvs := range ac.kvdatas {
		batch := ac.diskdb.NewBatch()
		for _, kv := range kvs {
			if kv.delete {
				batch.Delete(kv.key)
			} else {
				batch.Put(kv.key, kv.value)
			}
		}
		batch.Write()
		ac.clearChans <- kvs
	}
}

func (ac *ACProcessor) Clear() {
	for kvs := range ac.clearChans {
		ac.cache.Clear(kvs)
	}
}

func (ac *ACProcessor) GetDirty(key common.Hash) (*cachedNode, bool) {
	return ac.cache.GetDirty(key)
}

func (ac *ACProcessor) GetPreimage(key common.Hash) ([]byte, bool) {
	return ac.cache.GetPreimage(key)
}

func (ac *ACProcessor) SetDirty(key common.Hash, node *cachedNode) {
	ac.cache.SetDirty(key, node)
}

func (ac *ACProcessor) SetPreimages(images map[common.Hash][]byte) {
	ac.cache.SetPreimages(images)
}

type CacheList struct {
	caches []*Cache
}

func NewCacheList() *CacheList {
	cal := &CacheList{}
	cal.caches = make([]*Cache, CacheCount)
	for i := range cal.caches {
		cal.caches[i] = NewCache()
	}
	return cal
}

func (ac *CacheList) GetDirty(key common.Hash) (*cachedNode, bool) {
	h := xxhash.Sum64(key[:])
	idx := h % CacheCount
	return ac.caches[idx].GetDirty(key)
}

func (ac *CacheList) GetPreimage(key common.Hash) ([]byte, bool) {
	h := xxhash.Sum64(key[:])
	idx := h % CacheCount
	return ac.caches[idx].GetPreimage(key)
}

func (ac *CacheList) SetDirty(key common.Hash, node *cachedNode) {
	h := xxhash.Sum64(key[:])
	idx := h % CacheCount
	ac.caches[idx].SetDirty(key, node)
}

func (ac *CacheList) SetPreimages(images map[common.Hash][]byte) {
	for k, v := range images {
		h := xxhash.Sum64(k[:])
		idx := h % CacheCount
		ac.caches[idx].SetPreimage(k, v)
	}
}

func (ac *CacheList) Clear(kvs []*keyvalue) {
	for _, kv := range kvs {
		key := bytes.TrimPrefix(kv.key, rawdb.PreimagePrefix)
		h := xxhash.Sum64(key)
		idx := h % CacheCount
		hash := common.BytesToHash(key)
		ac.caches[idx].Clear(hash)
	}
	var lkvs = len(kvs)
	var ld = 0
	var lp = 0
	for _, ca := range ac.caches {
		sd, sp := ca.Size()
		ld += sd
		lp += sp
	}
	fmt.Println("#### end clear ####", "kvs", lkvs, "dirty size", ld,
		"preimages size", lp)
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

func (c *Cache) SetPreimage(key common.Hash, value []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.preimages[key] = value
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

func (c *Cache) Clear(hash common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if _, ok := c.dirties[hash]; ok {
		delete(c.dirties, hash)
	} else { // Need attention the preimages key preimageKey(hash)
		delete(c.preimages, hash)
	}
}

func (c *Cache) Size() (int, int) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.dirties), len(c.preimages)
}

func (c *Cache) Clears(kvs []*keyvalue) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, kv := range kvs {
		hash := common.BytesToHash(kv.key)
		if _, ok := c.dirties[hash]; ok {
			delete(c.dirties, hash)
		} else { // Need attention the preimages key preimageKey(hash)
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
	b.writes = nil
	b.size = 0
}

func (b *BatchEx) Write() error {
	if len(b.writes) == 0 {
		return nil
	}
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

func (db *Database) Close() {
	if db.acProcessor != nil {
		db.acProcessor.Close()
	}
}

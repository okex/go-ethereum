package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func genTestImages(n int) map[common.Hash][]byte {
	mp := make(map[common.Hash][]byte)
	for i := 0; i < n; i++ {
		mp[common.BytesToHash(randBytes(32))] = randBytes(32)
	}
	return mp
}

type testNode struct {
	key  []byte
	node *cachedNode
}

func genTestDirties(n int) []*testNode {
	var tnodes []*testNode
	for i := 0; i < n; i++ {
		tnodes = append(tnodes, &testNode{key: randBytes(32), node: &cachedNode{node: hashNode(randBytes(32))}})
	}
	return tnodes
}

func genKVs(n int) []*keyvalue {
	var kvs []*keyvalue
	for i := 0; i < n; i++ {
		var isdel bool
		value := randBytes(32)
		if rand.Int()%2 == 0 {
			isdel = true
			value = nil
		}
		kvs = append(kvs, &keyvalue{key: randBytes(32), value: value, delete: isdel})
	}
	return kvs
}

func TestCache(t *testing.T) {
	cache := NewCache()
	testImageCase := genTestImages(6)
	cache.SetPreimages(testImageCase)

	testDirtiesCase := genTestDirties(5)
	for _, ts := range testDirtiesCase {
		cache.SetDirty(common.BytesToHash(ts.key), ts.node)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, ts := range testDirtiesCase {
			ca, re := cache.GetDirty(common.BytesToHash(ts.key))
			assert.True(t, re)
			assert.Equal(t, ca, ts.node)
		}
	}()

	go func() {
		defer wg.Done()
		for k, v := range testImageCase {
			ca, re := cache.GetPreimage(k)
			assert.True(t, re)
			assert.Equal(t, ca, v)
		}
	}()
	wg.Wait()

	var kvs []*keyvalue
	for k := range testImageCase {
		key := make([]byte, len(k))
		copy(key, k[:])
		kvs = append(kvs, &keyvalue{key: key})
	}
	cache.Clears(kvs)
	assert.Equal(t, len(cache.preimages), 0)
	assert.NotEqual(t, len(cache.dirties), 0)

	for _, k := range testDirtiesCase {
		kvs = append(kvs, &keyvalue{key: k.key})
	}
	cache.Clears(kvs)
	assert.Equal(t, len(cache.preimages), 0)
	assert.Equal(t, len(cache.dirties), 0)
}

func TestACProcessor(t *testing.T) {
	diskdb := memorydb.New()
	acp := NewACProcessor(diskdb)

	batch := NewBatchEx(acp.kvdatas)

	kvs := genKVs(10)
	for _, kv := range kvs {
		if kv.delete {
			batch.Delete(kv.key)
		} else {
			batch.Put(kv.key, kv.value)
		}
	}
	assert.Equal(t, len(batch.writes), 10)
	batch.Write()
	batch.Reset()
	assert.Equal(t, batch.ValueSize(), 0)

	time.Sleep(time.Second)

	for _, kv := range kvs {
		v, err := diskdb.Get(kv.key)
		if kv.delete {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, kv.value, v)
		}
	}
}

func TestACIterator(t *testing.T) {
	triedb := NewDatabaseWithConfig(memorydb.New(), &Config{
		Preimages: true,
		EnableAC:  true,
	})
	ctr, _ := New(common.Hash{}, common.Hash{}, triedb)
	for _, val := range testdata1 {
		ctr.Update([]byte(val.k), []byte(val.v))
	}
	root, _, _ := ctr.Commit(false)

	triedb.Commit(root, true, nil)

	trie, err := New(root, common.Hash{}, triedb)
	assert.NoError(t, err)

	// should stop the ACCommit() for test
	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for _, kv := range testdata1 {
		if found[kv.k] != kv.v {
			t.Errorf("iterator value mismatch for %s: got %q want %q", kv.k, found[kv.v], kv.v)
		}
	}
}

func TestCacheList(t *testing.T) {
	preImages := genTestImages(1000)
	dirties := genTestDirties(1000)

	clist := NewCacheList()

	clist.SetPreimages(preImages)
	for _, n := range dirties {
		clist.SetDirty(common.BytesToHash(n.key), n.node)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, ts := range dirties {
			ca, re := clist.GetDirty(common.BytesToHash(ts.key))
			assert.True(t, re)
			assert.Equal(t, ca, ts.node)
		}
	}()

	go func() {
		defer wg.Done()
		for k, v := range preImages {
			ca, re := clist.GetPreimage(k)
			assert.True(t, re)
			assert.Equal(t, ca, v)
		}
	}()
	wg.Wait()

	// clear
	var kvs []*keyvalue
	for k := range preImages {
		key := make([]byte, len(k))
		copy(key, k[:])
		kvs = append(kvs, &keyvalue{key: key})
	}
	for _, k := range dirties {
		kvs = append(kvs, &keyvalue{key: k.key})
	}

	clist.Clear(kvs)
	// check
	for _, ca := range clist.caches {
		assert.Equal(t, len(ca.preimages), 0)
		assert.Equal(t, len(ca.dirties), 0)
	}
}

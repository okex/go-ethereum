package trie

import "sync/atomic"

type RuntimeState struct {
	nodeReadCount  int64
	cacheReadCount int64
}

func NewRuntimeState() *RuntimeState {
	return &RuntimeState{}
}

func (s *RuntimeState) addNodeReadCount() {
	atomic.AddInt64(&s.nodeReadCount, 1)
}

func (s *RuntimeState) getNodeReadCount() int {
	return int(atomic.LoadInt64(&s.nodeReadCount))
}

func (s *RuntimeState) addCacheReadCount() {
	atomic.AddInt64(&s.cacheReadCount, 1)
}

func (s *RuntimeState) getCacheReadCount() int {
	return int(atomic.LoadInt64(&s.cacheReadCount))
}

func (s *RuntimeState) resetCount() {
	atomic.StoreInt64(&s.nodeReadCount, 0)
	atomic.StoreInt64(&s.cacheReadCount, 0)
}

func (db *Database) GetNodeReadCount() int {
	return db.statistics.getNodeReadCount()
}

func (db *Database) GetCacheReadCount() int {
	return db.statistics.getCacheReadCount()
}

func (db *Database) ResetCount() {
	db.statistics.resetCount()
}

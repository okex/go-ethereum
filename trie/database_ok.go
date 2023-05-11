package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/snap"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func (db *Database) UpdateForOK(root common.Hash, parent common.Hash, nodes *MergedNodeSet) error {
	if db.preimages != nil {
		db.preimages.commit(false)
	}
	if snap, ok := db.backend.(*snap.Database); ok {
		sets := make(map[common.Hash]map[string]*trienode.WithPrev)
		for owner, set := range nodes.sets {
			sets[owner] = set.nodes
		}
		return snap.Update(root, parent, sets)
	}
	return db.backend.(*hashDatabase).Update(root, parent, nodes)
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.Database {
	return db.diskdb
}

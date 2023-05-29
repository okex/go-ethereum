package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/snap"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func (db *Database) UpdateForOK(root common.Hash, parent common.Hash, nodes *MergedNodeSet, accRetrieval func([]byte) common.Hash) error {
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
	return db.backend.(*hashDatabase).UpdateForOK(nodes, accRetrieval)
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.Database {
	return db.diskdb
}

// UpdateForOK inserts the dirty nodes in provided nodeset into database and
// link the account trie with multiple storage tries if necessary.
func (db *hashDatabase) UpdateForOK(nodes *MergedNodeSet, accRetrieval func([]byte) common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Insert dirty nodes into the database. In the same tree, it must be
	// ensured that children are inserted first, then parent so that children
	// can be linked with their parent correctly.
	//
	// Note, the storage tries must be flushed before the account trie to
	// retain the invariant that children go into the dirty cache first.
	var order []common.Hash
	for owner := range nodes.sets {
		if owner == (common.Hash{}) {
			continue
		}
		order = append(order, owner)
	}
	if _, ok := nodes.sets[common.Hash{}]; ok {
		order = append(order, common.Hash{})
	}
	for _, owner := range order {
		subset := nodes.sets[owner]
		subset.forEachWithOrder(func(path string, n *trienode.Node) {
			if n.IsDeleted() {
				return // ignore deletion
			}
			db.insert(n.Hash, n.Blob)
		})
	}
	// Link up the account trie and storage trie if the node points
	// to an account trie leaf.
	if set, present := nodes.sets[common.Hash{}]; present {
		for _, n := range set.leaves {
			storageRoot := accRetrieval(n.blob)
			if storageRoot != types.EmptyRootHash && storageRoot != (common.Hash{}) {
				db.reference(storageRoot, n.parent)
			}
		}
	}
	return nil
}

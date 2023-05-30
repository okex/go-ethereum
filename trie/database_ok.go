package trie

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)

func (db *Database) UpdateForOK(nodes *MergedNodeSet, accRetrieval func([]byte) common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	for k, v := range nodes.sets {
		fmt.Printf("======set-k:%x\n", k)
		for k, v := range v.nodes {
			fmt.Printf("node-k:%x\nnodeHash:%x\n", k, v.hash)
			nodeToBytes(v.node)
		}
	}

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
		for _, path := range subset.paths {
			n, ok := subset.nodes[path]
			if !ok {
				return fmt.Errorf("missing node %x %v", owner, path)
			}
			if fmt.Sprintf("%x", n.hash.Bytes()) == "b3f7320c2d56c831857ba456fa234d282e1f395666d80414999181b3fd49b478" {
				fmt.Println("to insert==================")
				nodeToBytes(n.node)
			}
			db.insert(n.hash, int(n.size), n.node)
		}
	}
	// Link up the account trie and storage trie if the node points
	// to an account trie leaf.
	if set, present := nodes.sets[common.Hash{}]; present {
		for _, n := range set.leaves {
			storageRoot := accRetrieval(n.blob)
			if storageRoot != emptyRoot && storageRoot != (common.Hash{}) {
				db.reference(storageRoot, n.parent)
			}
		}

	}

	return nil
}

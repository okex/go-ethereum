package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/go-amino"
)

type MptDeltaMap map[string]*MptDelta

type MptDelta struct {
	NodeDelta []*NodeDelta    `json:"node_delta"`
	Storage   []*StorageDelta `json:"storage"`
}

type StorageDelta struct {
	Addr      common.Address `json:"addr"`
	PreAcc    []byte         `json:"pre_acc"`
	NodeDelta []*NodeDelta   `json:"node_delta"`
}

type NodeDelta struct {
	Key string `json:"key"`
	Val []byte `json:"val"`
}

// TreeDeltaMapImp convert map[string]*TreeDelta to struct
type TreeDeltaMapImp struct {
	Key       string
	TreeValue *MptDelta
}

func (mdm MptDeltaMap) Marshal() []byte {
	mptDeltaSlice := make([]*TreeDeltaMapImp, 0, len(mdm))
	for k, v := range mdm {
		mptDeltaSlice = append(mptDeltaSlice, &TreeDeltaMapImp{k, v})
	}

	cdc := amino.NewCodec()
	return cdc.MustMarshalBinaryBare(mptDeltaSlice)
}

func (mdm MptDeltaMap) Unmarshal(deltaBytes []byte) error {
	var mptDeltaSlice []*TreeDeltaMapImp
	cdc := amino.NewCodec()
	if err := cdc.UnmarshalBinaryBare(deltaBytes, &mptDeltaSlice); err != nil {
		return err
	}
	for _, delta := range mptDeltaSlice {
		mdm[delta.Key] = delta.TreeValue
	}
	return nil
}

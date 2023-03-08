package trie

import (
	"github.com/tendermint/go-amino"
)

type MptDeltaMap map[string]*MptDelta

type MptDelta struct {
	NodeDelta []*NodeDelta `json:"node_delta"`
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

/*
func (tdm MptDeltaMap) MarshalAmino() ([]*TreeDeltaMapImp, error) {
	keys := make([]string, len(tdm))
	index := 0
	for k := range tdm {
		keys[index] = k
		index++
	}
	sort.Strings(keys)

	treeDeltaList := make([]*TreeDeltaMapImp, len(tdm))
	index = 0
	for _, k := range keys {
		treeDeltaList[index] = &TreeDeltaMapImp{Key: k, TreeValue: tdm[k]}
		index++
	}
	return treeDeltaList, nil
}
func (tdm MptDeltaMap) UnmarshalAmino(treeDeltaList []*TreeDeltaMapImp) error {
	for _, v := range treeDeltaList {
		tdm[v.Key] = v.TreeValue
	}
	return nil
}

// MarshalToAmino marshal to amino bytes
func (tdm MptDeltaMap) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	fieldKeysType := byte(1<<3 | 2)

	if len(tdm) == 0 {
		return buf.Bytes(), nil
	}

	keys := make([]string, len(tdm))
	index := 0
	for k := range tdm {
		keys[index] = k
		index++
	}
	sort.Strings(keys)

	// encode a pair of data one by one
	for _, k := range keys {
		err := buf.WriteByte(fieldKeysType)
		if err != nil {
			return nil, err
		}

		// map must convert to new struct before it marshal
		ti := &TreeDeltaMapImp{Key: k, TreeValue: tdm[k]}
		data, err := ti.MarshalToAmino(cdc)
		if err != nil {
			return nil, err
		}
		// write marshal result to buffer
		err = amino.EncodeByteSliceToBuffer(&buf, data)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// TreeDeltaMapImp convert map[string]*TreeDelta to struct
type TreeDeltaMapImp struct {
	Key       string
	TreeValue *MptDelta
}

// MarshalToAmino marshal data to amino bytes
func (ti *TreeDeltaMapImp) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	if ti == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	fieldKeysType := [2]byte{1<<3 | 2, 2<<3 | 2}
	for pos := 1; pos <= 2; pos++ {
		switch pos {
		case 1:
			if len(ti.Key) == 0 {
				break
			}
			err := buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}

			err = amino.EncodeStringToBuffer(&buf, ti.Key)
			if err != nil {
				return nil, err
			}
		case 2:
			if ti.TreeValue == nil {
				break
			}
			err := buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}
			var data []byte
			data, err = ti.TreeValue.MarshalToAmino(cdc)
			if err != nil {
				return nil, err
			}

			err = amino.EncodeByteSliceToBuffer(&buf, data)
			if err != nil {
				return nil, err
			}

		default:
			panic("unreachable")
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalFromAmino unmarshal data from amino bytes.
func (ti *TreeDeltaMapImp) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]
		if len(data) == 0 {
			break
		}
		pos, pbType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if pbType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, _ = amino.DecodeUvarint(data)

			data = data[n:]
			if len(data) < int(dataLen) {
				return errors.New("not enough data")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			ti.Key = string(subData)

		case 2:
			tv := &MptDelta{}
			if len(subData) != 0 {
				err := tv.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			ti.TreeValue = tv

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

func (td *MptDelta) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	const pbKey = 1<<3 | 2
	//encode data
	for _, node := range td.NodeDelta {
		err := buf.WriteByte(pbKey)
		if err != nil {
			return nil, err
		}

		var data []byte
		if node != nil {
			data, err = node.MarshalToAmino(cdc)
			if err != nil {
				return nil, err
			}
		}
		err = amino.EncodeByteSliceToBuffer(&buf, data)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
func (td *MptDelta) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]
		if len(data) == 0 {
			break
		}
		pos, pbType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if pbType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, _ = amino.DecodeUvarint(data)

			data = data[n:]
			if len(data) < int(dataLen) {
				return errors.New("not enough data")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			var ni *NodeJsonImp = nil
			if len(subData) != 0 {
				ni = &NodeJsonImp{}
				err := ni.UnmarshalFromAmino(cdc, subData)
				if err != nil {
					return err
				}
			}
			td.NodeDelta = append(td.NodeDelta, ni)

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}

type NodeJsonImp struct {
	Key       string
	NodeValue *NodeDelta
}

// MarshalToAmino marshal data to amino bytes.
func (ni *NodeJsonImp) MarshalToAmino(cdc *amino.Codec) ([]byte, error) {
	var buf bytes.Buffer
	fieldKeysType := [2]byte{1<<3 | 2, 2<<3 | 2}
	for pos := 1; pos <= 2; pos++ {
		switch pos {
		case 1:
			if len(ni.Key) == 0 {
				break
			}
			err := buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}

			err = amino.EncodeStringToBuffer(&buf, ni.Key)
			if err != nil {
				return nil, err
			}
		case 2:
			if ni.NodeValue == nil {
				break
			}
			err := buf.WriteByte(fieldKeysType[pos-1])
			if err != nil {
				return nil, err
			}
			var data []byte
			data, err = ni.NodeValue.MarshalToAmino(cdc)
			if err != nil {
				return nil, err
			}

			err = amino.EncodeByteSliceToBuffer(&buf, data)
			if err != nil {
				return nil, err
			}

		default:
			panic("unreachable")
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalFromAmino unmarshal data from amino bytes.
func (ni *NodeJsonImp) UnmarshalFromAmino(cdc *amino.Codec, data []byte) error {
	var dataLen uint64 = 0
	var subData []byte

	for {
		data = data[dataLen:]
		if len(data) == 0 {
			break
		}
		pos, pbType, err := amino.ParseProtoPosAndTypeMustOneByte(data[0])
		if err != nil {
			return err
		}
		data = data[1:]

		if pbType == amino.Typ3_ByteLength {
			var n int
			dataLen, n, _ = amino.DecodeUvarint(data)

			data = data[n:]
			if len(data) < int(dataLen) {
				return errors.New("not enough data")
			}
			subData = data[:dataLen]
		}

		switch pos {
		case 1:
			ni.Key = string(subData)

		case 2:
			nj := &NodeDelta{}
			if len(subData) != 0 {
				err := cdc.UnmarshalBinaryBare(subData, nj)
				if err != nil {
					return err
				}
			}
			ni.NodeValue = nj

		default:
			return fmt.Errorf("unexpect feild num %d", pos)
		}
	}
	return nil
}
*/

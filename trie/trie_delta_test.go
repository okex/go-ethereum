package trie

import (
	"github.com/influxdata/influxdb/pkg/testing/assert"
	"testing"
)

func TestMptDeltaMapMarshalAndUnmarshal(t *testing.T) {
	nodeDelta := make([]*NodeDelta, 1)
	nodeDelta[0] = &NodeDelta{Key: "test-key", Val: []byte("test-val")}
	tests := []struct {
		name string
		mdm  MptDeltaMap
	}{
		// TODO: Add test cases.
		{"normal", MptDeltaMap{"test1": &MptDelta{nodeDelta}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outBytes := tt.mdm.Marshal()
			outDelta := MptDeltaMap{}
			outDelta.Unmarshal(outBytes)
			assert.Equal(t, tt.mdm, outDelta)
		})
	}
}

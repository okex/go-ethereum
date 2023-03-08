package contractverifier

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

type callInfo struct {
	Dept            int64
	CallType        string
	Name            string
	From            string
	To              string
	Input           string
	Output          string
	Value           string
	ValueWei        string
	FromNonce       uint64
	Create2Salt     string
	Create2CodeHash string
}

func newCallInfo(depth int64, callType, name, from, to, input, output, value, valueWei string, fromNonce uint64, salt, codehash string) callInfo {
	return callInfo{
		Dept:            depth,
		CallType:        callType,
		Name:            name,
		From:            from,
		To:              to,
		Input:           input,
		Output:          output,
		Value:           value,
		ValueWei:        valueWei,
		FromNonce:       fromNonce,
		Create2Salt:     salt,
		Create2CodeHash: codehash,
	}
}

func (ct callTracerTest) getCallInfoArray(t *testing.T) []callInfo {
	return ct.Result.getCallInfoArray(0, "", t)

}

func (ct callTrace) getCallInfoArray(depth int, nameSuffix string, t *testing.T) []callInfo {
	results := make([]callInfo, 0)

	op := vm.StringToOp(ct.Type)

	callType := ""
	from := ct.From.String()
	to := ct.To.String()
	valueWei := ""
	if depth != 0 {
		callType = convertInnertxType(op, t)
		from = ct.From.Hash().String()
		to = ct.To.Hash().String()
		valueWei = ct.Value.ToInt().Text(10)
	}

	if ct.Value.ToInt().Int64() != 0 && len(ct.Error) == 0 {
		//if op == vm.SELFDESTRUCT {
		//valueWei = ct.Value.ToInt().Text(10)
		//} else {
		//	value = v
		//}
	}
	ci := newCallInfo(int64(depth), callType, callType+nameSuffix, from, to, "", "", "", valueWei, ct.FromNonce, ct.Create2Salt, ct.Create2CodeHash)
	results = append(results, ci)

	for i, _ := range ct.Calls {
		nextNameSuffix := nameSuffix + "_" + strconv.Itoa(i)
		nextResults := ct.Calls[i].getCallInfoArray(depth+1, nextNameSuffix, t)
		results = append(results, nextResults...)
	}

	return results
}

func convertInnertxType(op vm.OpCode, t *testing.T) string {
	txtype := ""
	switch op {
	case vm.CREATE:
		txtype = "create"
	case vm.CREATE2:
		txtype = "create2"
	case vm.CALL:
		txtype = "call"
	case vm.CALLCODE:
		txtype = "callcode"
	case vm.STATICCALL:
		txtype = "staticcall"
	case vm.DELEGATECALL:
		txtype = "delegatecall"
	case vm.SELFDESTRUCT:
		txtype = "suicide"
	default:
		t.Fatalf("unsupported opcode for innertx: %v\n", op)
	}
	return txtype
}

func innertxsToCallInfo(innertxs []*vm.InnerTx) []callInfo {
	results := make([]callInfo, 0)

	for _, innertx := range innertxs {
		ci := newCallInfo(innertx.Dept.Int64(), innertx.CallType, innertx.Name, innertx.From, innertx.To, innertx.Input, innertx.Output, innertx.Value, innertx.ValueWei, innertx.FromNonce, innertx.Create2Salt, innertx.Create2CodeHash)
		results = append(results, ci)
	}
	return results
}

func TestOKCInnerTx(t *testing.T) {
	testdir := filepath.Join("testdata", "innertx")
	files, err := ioutil.ReadDir(testdir)
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}

	for _, file := range files {
		if file.Name() == "inner_create_oog_outer_throw.json" ||
			file.Name() == "inner_instafail.json" {
			// txs in those two file will execute error, and innertx will record
			// error execution(with `InnerTx.Error` contains error message),
			// but there's no error execution information in those json files,
			// so we ignore those files here, and test them in TestOKCInnerTxError.
			continue
		}

		file := file // capture range variable
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			blob, err := ioutil.ReadFile(filepath.Join(testdir, file.Name()))
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			test := new(callTracerTest)
			if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}

			tx := new(types.Transaction)
			if err := rlp.DecodeBytes(common.FromHex(test.Input), tx); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}
			tr := newTxRunner(test.Genesis, test.Context)
			evm := tr.executeTx(vm.Config{}, tx, t)

			expect := test.getCallInfoArray(t)
			actual := innertxsToCallInfo(evm.InnerTxies)
			assert.Equal(t, expect, actual, "testing file %s", file.Name())
		})
	}
}

func TestOKCInnerTxError(t *testing.T) {
	tests := []struct {
		filename       string
		extraCallinfos []callInfo
	}{
		// TODO: Add test cases.
		{
			filename: "inner_create_oog_outer_throw.json",
			extraCallinfos: []callInfo{
				{
					Dept:      1,
					CallType:  "create",
					Name:      "create_0",
					From:      "0x0000000000000000000000001d3ddf7caf024f253487e18bc4a15b1a360c170a",
					To:        "0x0000000000000000000000005cb4a6b902fcb21588c86c3517e797b07cdaadb9",
					Input:     "",
					Output:    "",
					Value:     "",
					ValueWei:  "0",
					FromNonce: 789,
				},
			},
		},
		{
			filename: "inner_instafail.json",
			extraCallinfos: []callInfo{
				{
					Dept:     1,
					CallType: "call",
					Name:     "call_0",
					From:     "0x0000000000000000000000006c06b16512b332e6cd8293a2974872674716ce18",
					To:       "0x00000000000000000000000066fdfd05e46126a07465ad24e40cc0597bc1ef31",
					Input:    "",
					Output:   "",
					Value:    "",
					ValueWei: "1500000000000000000",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(strings.TrimSuffix(tt.filename, ".json"), func(t *testing.T) {
			blob, err := ioutil.ReadFile(filepath.Join("testdata", "innertx", tt.filename))
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			test := new(callTracerTest)
			if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}

			tx := new(types.Transaction)
			if err := rlp.DecodeBytes(common.FromHex(test.Input), tx); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}
			tr := newTxRunner(test.Genesis, test.Context)
			evm := tr.executeTx(vm.Config{}, tx, t)

			expect := append(test.getCallInfoArray(t), tt.extraCallinfos...)
			actual := innertxsToCallInfo(evm.InnerTxies)
			assert.Equal(t, expect, actual, "testing file %s", tt.filename)
		})
	}
}

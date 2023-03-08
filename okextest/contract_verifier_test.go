package contractverifier

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
)

const (
	UNKNOWN = iota
	SELFDESTRUCT
	NORMAL_CALL
)

type contractVerifierTest struct {
	verifiers []verifierTest
}

func NewContractVerifier() *contractVerifierTest {
	return &contractVerifierTest{verifiers: make([]verifierTest, 0)}
}

func (cv *contractVerifierTest) Verify(stateDB vm.StateDB, op vm.OpCode, from, to common.Address, input []byte, value *big.Int) error {
	verifierTest := NewVerifierTest(UNKNOWN, op, from, to, input, value)
	if op == vm.SELFDESTRUCT {
		verifierTest.VerifierType = SELFDESTRUCT
	} else if op == vm.CALL || op == vm.DELEGATECALL || op == vm.STATICCALL || op == vm.CALLCODE {
		verifierTest.VerifierType = NORMAL_CALL
	}
	cv.verifiers = append(cv.verifiers, verifierTest)
	return nil
}

type verifierTest struct {
	VerifierType int
	Op           vm.OpCode
	From         common.Address
	To           common.Address
	Input        string
	Value        string
}

func NewVerifierTest(verifierType int, op vm.OpCode, from, to common.Address, input []byte, value *big.Int) verifierTest {
	return verifierTest{
		VerifierType: verifierType,
		Op:           op,
		From:         from,
		To:           to,
		Input:        hexutil.Encode(input),
		Value:        fmt.Sprintf("0x%s", value.Text(16)),
	}
}

func convertVerifierType(op vm.OpCode) int {
	result := UNKNOWN
	if op == vm.SELFDESTRUCT {
		result = SELFDESTRUCT
	} else if op == vm.CALL || op == vm.DELEGATECALL || op == vm.STATICCALL || op == vm.CALLCODE {
		result = NORMAL_CALL
	}
	return result
}

func (ct *callTrace) GetArrayCall() []verifierTest {
	results := make([]verifierTest, 0)
	self := NewVerifierTest(convertVerifierType(vm.StringToOp(ct.Type)), vm.StringToOp(ct.Type), ct.From, ct.To, nil, nil)
	self.Input = ct.Input.String()
	self.Value = ct.Value.String()
	results = append(results, self)

	callsNumber := len(ct.Calls)
	if callsNumber == 0 {
		return results
	}

	for i, _ := range ct.Calls {
		nextResults := ct.Calls[i].GetArrayCall()
		results = append(results, nextResults...)
	}
	return results
}

func (ct callTracerTest) GetArrayResult() []verifierTest {
	return ct.Result.GetArrayCall()
}

// Iterates over all the input-output datasets in the tracer test harness and
// runs the JavaScript tracers against them.
func TestOKVerify(t *testing.T) {
	testdir := filepath.Join("testdata", "innertx")
	files, err := ioutil.ReadDir(testdir)
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}

	for _, file := range files {
		file := file // capture range variable
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			// Call tracer test found, read if from disk
			blob, err := ioutil.ReadFile(filepath.Join(testdir, file.Name()))
			if err != nil {
				t.Fatalf("failed to read testcase: %v", err)
			}
			test := new(callTracerTest)
			if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}

			tx := new(types.Transaction)
			if err := rlp.DecodeBytes(common.FromHex(test.Input), tx); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}

			verifierTest := NewContractVerifier()
			tr := newTxRunner(test.Genesis, test.Context)
			tr.executeTx(vm.Config{ContractVerifier: verifierTest}, tx, t)

			actual := verifierTest.verifiers
			excepted := test.GetArrayResult()
			require.Equal(t, excepted, actual)
		})
	}
}

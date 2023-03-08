package contractverifier

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"math/big"
	"strings"
	"testing"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// callTracerTest defines a single test to check the call tracer against.
type callTracerTest struct {
	Genesis *core.Genesis `json:"genesis"`
	Context *callContext  `json:"context"`
	Input   string        `json:"input"`
	Result  *callTrace    `json:"result"`
}

type callContext struct {
	Number     math.HexOrDecimal64   `json:"number"`
	Difficulty *math.HexOrDecimal256 `json:"difficulty"`
	Time       math.HexOrDecimal64   `json:"timestamp"`
	GasLimit   math.HexOrDecimal64   `json:"gasLimit"`
	Miner      common.Address        `json:"miner"`
}

// callTrace is the result of a callTracer run.
type callTrace struct {
	Type            string          `json:"type"`
	From            common.Address  `json:"from"`
	To              common.Address  `json:"to"`
	Input           hexutil.Bytes   `json:"input"`
	Output          hexutil.Bytes   `json:"output"`
	Gas             *hexutil.Uint64 `json:"gas,omitempty"`
	GasUsed         *hexutil.Uint64 `json:"gasUsed,omitempty"`
	Value           *hexutil.Big    `json:"value,omitempty"`
	Error           string          `json:"error,omitempty"`
	Calls           []callTrace     `json:"calls,omitempty"`
	FromNonce       uint64          `json:"fromNonce,omitempty"`
	Create2Salt     string          `json:"create2Salt,omitempty"`
	Create2CodeHash string          `json:"create2CodeHash,omitempty"`
}

type txRunner struct {
	genesis *core.Genesis
	callCtx *callContext
	statedb *state.StateDB
}

func newTxRunner(genesis *core.Genesis, callCtx *callContext) *txRunner {
	_, statedb := makePreState(rawdb.NewMemoryDatabase(), genesis.Alloc, false)
	return &txRunner{
		genesis: genesis,
		callCtx: callCtx,
		statedb: statedb,
	}
}

func (tr *txRunner) executeTx(evmConfig vm.Config, tx *types.Transaction, t *testing.T) *vm.EVM {
	signer := types.MakeSigner(tr.genesis.Config, new(big.Int).SetUint64(uint64(tr.callCtx.Number)))
	origin, _ := signer.Sender(tx)
	txContext := vm.TxContext{
		Origin:   origin,
		GasPrice: tx.GasPrice(),
	}
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    tr.callCtx.Miner,
		BlockNumber: new(big.Int).SetUint64(uint64(tr.callCtx.Number)),
		Time:        new(big.Int).SetUint64(uint64(tr.callCtx.Time)),
		Difficulty:  (*big.Int)(tr.callCtx.Difficulty),
		GasLimit:    uint64(tr.callCtx.GasLimit),
		BaseFee:     big.NewInt(100),
	}

	evm := vm.NewEVM(context, txContext, tr.statedb, tr.genesis.Config, evmConfig)

	msg, err := tx.AsMessage(signer, nil)
	if err != nil {
		t.Fatalf("failed to prepare transaction for tracing: %v", err)
	}
	st := core.NewStateTransition(evm, msg, new(core.GasPool).AddGas(tx.Gas()))
	if _, err = st.TransitionDb(); err != nil {
		t.Fatalf("failed to execute transaction: %v", err)
	}

	return evm

}

func makePreState(db ethdb.Database, accounts core.GenesisAlloc, snapshotter bool) (*snapshot.Tree, *state.StateDB) {
	sdb := state.NewDatabase(db)
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		statedb.SetBalance(addr, a.Balance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(false)

	var snaps *snapshot.Tree
	if snapshotter {
		snaps, _ = snapshot.New(db, sdb.TrieDB(), 1, root, false, true, false)
	}
	statedb, _ = state.New(root, sdb, snaps)
	return snaps, statedb
}

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}

func buildSignedTx(prvKey, toAddress string, nonce uint64, value int64 /*in wei*/, data []byte) (*types.Transaction, error) {
	const (
		defaultChainID = 3
		//value    = 0 /*in wei*/
		defaultGasLimit = 5000000
		defaultGasPrice = 21000
	)
	privateKey, err := crypto.HexToECDSA(prvKey)
	if err != nil {
		panic(err)
	}

	var to *common.Address = nil
	if len(toAddress) != 0 {
		addr := common.HexToAddress(toAddress)
		to = &addr
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    big.NewInt(value),
		Gas:      defaultGasLimit,
		GasPrice: big.NewInt(defaultGasPrice),
		Data:     data,
	})
	return types.SignTx(tx, types.NewEIP155Signer(big.NewInt(defaultChainID)), privateKey)
}

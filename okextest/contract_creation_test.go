package contractverifier

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

type initInfo struct {
	Genesis *core.Genesis `json:"genesis"`
	Context *callContext  `json:"context"`
}

func TestInnerTxContractCreate(t *testing.T) {
	const (
		// eth_address: 0xbbE4733d85bc2b90682147779DA49caB38C0aA1F
		prvKey1 = "8ff3ca2d9985c3a52b459e2f6e7822b23e1af845961e22128d5f372fb9aa5f17"
		// eth address: 0x4ff850ee577167E8321cb13bC95E19f512A56ba0
		prvKey2 = "9ff3ca2d9985c3a52b459e2f6e7822b23e1af845961e22128d5f372fb9aa5f17"
	)
	address1 := addressByPrivateKey(prvKey1, t)
	address2 := addressByPrivateKey(prvKey2, t)

	factoryA := getCreateFactoryAInfo()
	factoryB := getCreateFactoryBInfo()
	factoryC := getCreateFactoryCInfo()
	bank := getBankInfo()

	blob, err := ioutil.ReadFile(filepath.Join("testdata", "contract_creation", "init_config.json"))
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	iinfo := new(initInfo)
	if err := json.Unmarshal(blob, iinfo); err != nil {
		t.Fatalf("failed to parse testcase: %v", err)
	}
	_, ok := iinfo.Genesis.Alloc[address1]
	assert.True(t, ok, "address %s must in genesis", address1.String())
	_, ok = iinfo.Genesis.Alloc[address2]
	assert.True(t, ok, "address %s must in genesis", address2.String())

	tr := newTxRunner(iinfo.Genesis, iinfo.Context)

	// test deploy createFactoryA
	nonceBeforeTx := tr.statedb.GetNonce(address1)
	tx, err := buildSignedTx(prvKey1, "", nonceBeforeTx, 0, factoryA.code)
	assert.NoError(t, err)
	evm := tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 1, len(evm.InnerTxies))
	factoryA.addr = common.HexToAddress(evm.InnerTxies[0].To)
	expect1 := createInnerTx(0, "", "", address1.String(), factoryA.addr.String(), 0, "", nonceBeforeTx, nil, nil)
	assert.Equal(t, expect1, evm.InnerTxies[0])

	// test createFactoryA.createB
	nonceBeforeTx = tr.statedb.GetNonce(address2)
	factoryANonce := tr.statedb.GetNonce(factoryA.addr)
	tx, err = buildSignedTx(prvKey2, factoryA.addr.String(), tr.statedb.GetNonce(address2), 0, factoryA.methodID["createB"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	factoryB.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryA.addr.String(), 0, "", 0, nil, nil)
	expect2 := createInnerTx(1, "create", "create_0", factoryA.addr.Hash().String(), factoryB.addr.Hash().String(), 0x4a497e, "0", factoryANonce, nil, nil)
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])

	// test createFactoryB.createC
	nonceBeforeTx = tr.statedb.GetNonce(address2)
	factoryBNonce := tr.statedb.GetNonce(factoryB.addr)
	tx, err = buildSignedTx(prvKey2, factoryB.addr.String(), tr.statedb.GetNonce(address2), 0, factoryB.methodID["createC"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	factoryC.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryB.addr.String(), 0, "", 0, nil, nil)
	expect2 = createInnerTx(1, "create", "create_0", factoryB.addr.Hash().String(), factoryC.addr.Hash().String(), 0x4a4a60, "0", factoryBNonce, nil, nil)
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])

	// test createFactoryC.createBank
	nonceBeforeTx = tr.statedb.GetNonce(address2)
	factoryCNonce := tr.statedb.GetNonce(factoryC.addr)
	tx, err = buildSignedTx(prvKey2, factoryC.addr.String(), tr.statedb.GetNonce(address2), 0, factoryC.methodID["createBank"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	bank.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryC.addr.String(), 0, "", 0, nil, nil)
	expect2 = createInnerTx(1, "create", "create_0", factoryC.addr.Hash().String(), bank.addr.Hash().String(), 0x4a4c12, "0", factoryCNonce, nil, nil)
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])
}

func TestInnerTxContractCreate2(t *testing.T) {
	const (
		// eth_address: 0xbbE4733d85bc2b90682147779DA49caB38C0aA1F
		prvKey1 = "8ff3ca2d9985c3a52b459e2f6e7822b23e1af845961e22128d5f372fb9aa5f17"
		// eth address: 0x4ff850ee577167E8321cb13bC95E19f512A56ba0
		prvKey2 = "9ff3ca2d9985c3a52b459e2f6e7822b23e1af845961e22128d5f372fb9aa5f17"
	)
	address1 := addressByPrivateKey(prvKey1, t)
	address2 := addressByPrivateKey(prvKey2, t)

	factoryA := getCreate2FactoryAInfo()
	factoryB := getCreate2FactoryBInfo()
	factoryC := getCreate2FactoryCInfo()
	bank2 := getBank2Info()

	blob, err := ioutil.ReadFile(filepath.Join("testdata", "contract_creation", "init_config.json"))
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	iinfo := new(initInfo)
	if err := json.Unmarshal(blob, iinfo); err != nil {
		t.Fatalf("failed to parse testcase: %v", err)
	}
	_, ok := iinfo.Genesis.Alloc[address1]
	assert.True(t, ok, "address %s must in genesis", address1.String())
	_, ok = iinfo.Genesis.Alloc[address2]
	assert.True(t, ok, "address %s must in genesis", address2.String())

	tr := newTxRunner(iinfo.Genesis, iinfo.Context)

	// test deploy create2FactoryA
	nonceBeforeTx := tr.statedb.GetNonce(address1)
	tx, err := buildSignedTx(prvKey1, "", nonceBeforeTx, 0, factoryA.code)
	assert.NoError(t, err)
	evm := tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 1, len(evm.InnerTxies))
	factoryA.addr = common.HexToAddress(evm.InnerTxies[0].To)
	expect1 := createInnerTx(0, "", "", address1.String(), factoryA.addr.String(), 0, "", nonceBeforeTx, nil, nil)
	assert.Equal(t, expect1, evm.InnerTxies[0])

	// test create2FactoryA.createB
	tx, err = buildSignedTx(prvKey2, factoryA.addr.String(), tr.statedb.GetNonce(address2), 0, factoryA.methodID["createB"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	factoryB.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryA.addr.String(), 0, "", 0, nil, nil)
	expect2 := createInnerTx(1, "create2", "create2_0", factoryA.addr.Hash().String(), factoryB.addr.Hash().String(), 0x4a4762, "0", 0, factoryA.create2Salt[:], factoryB.codehash(&address2))
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])

	// test create2FactoryB.createC
	tx, err = buildSignedTx(prvKey2, factoryB.addr.String(), tr.statedb.GetNonce(address2), 0, factoryB.methodID["createC"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	factoryC.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryB.addr.String(), 0, "", 0, nil, nil)
	expect2 = createInnerTx(1, "create2", "create2_0", factoryB.addr.Hash().String(), factoryC.addr.Hash().String(), 0x4a4918, "0", 0, factoryB.create2Salt[:], factoryC.codehash(&address2))
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])

	// test create2FactoryC.createBank
	tx, err = buildSignedTx(prvKey2, factoryC.addr.String(), tr.statedb.GetNonce(address2), 0, factoryC.methodID["createBank"])
	assert.NoError(t, err)
	evm = tr.executeTx(vm.Config{}, tx, t)
	assert.Equal(t, 2, len(evm.InnerTxies))
	bank2.addr = common.HexToAddress(evm.InnerTxies[1].To)
	expect1 = createInnerTx(0, "", "", address2.String(), factoryC.addr.String(), 0, "", 0, nil, nil)
	expect2 = createInnerTx(1, "create2", "create2_0", factoryC.addr.Hash().String(), bank2.addr.Hash().String(), 0x4a4baa, "0", 0, factoryC.create2Salt[:], bank2.codehash(nil))
	assert.Equal(t, expect1, evm.InnerTxies[0])
	assert.Equal(t, expect2, evm.InnerTxies[1])
}

func addressByPrivateKey(prvKey string, t *testing.T) common.Address {
	privateKey, err := crypto.HexToECDSA(prvKey)
	assert.NoError(t, err)
	return crypto.PubkeyToAddress(privateKey.PublicKey)
}

func createInnerTx(depth int64, callType, name, from, to string, gasUsed uint64, valueWei string, fromNonce uint64, salt, codehash []byte) *vm.InnerTx {
	saltStr := ""
	if salt != nil {
		saltStr = hex.EncodeToString(salt)
	}
	codehashStr := ""
	if codehash != nil {
		codehashStr = hex.EncodeToString(codehash)
	}
	return &vm.InnerTx{
		InnerTxBasic: vm.InnerTxBasic{
			Dept:          *big.NewInt(depth),
			InternalIndex: *big.NewInt(0),
			CallType:      callType,
			Name:          name,
			TraceAddress:  "",
			CodeAddress:   "",
			From:          from,
			To:            to,
			Input:         "",
			Output:        "",
			IsError:       false,
			GasUsed:       gasUsed,
			Value:         "",
			ValueWei:      valueWei,
			Error:         "",
		},
		FromNonce:       fromNonce,
		Create2Salt:     saltStr,
		Create2CodeHash: codehashStr,
	}
}

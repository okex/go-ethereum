package vm

import "math/big"

const (
	// TODO: unified constant in exchain/libs/cosmos-sdk/types/innertx/innerTx.go
	CosmosCallType = "cosmos"
	EvmCreateName  = "cosmos-create"
)

// InnerTxBasic stores the basic field of a innertx.
// NOTE: DON'T change this struct for:
// 1. It will be written to database, and must be keep the same type When reading history data from db
// 2. It will be returned by rpc method
type InnerTxBasic struct {
	Dept          big.Int `json:"dept"`
	InternalIndex big.Int `json:"internal_index"`
	CallType      string  `json:"call_type"`
	Name          string  `json:"name"`
	TraceAddress  string  `json:"trace_address"`
	CodeAddress   string  `json:"code_address"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	Input         string  `json:"input"`
	Output        string  `json:"output"`
	IsError       bool    `json:"is_error"`
	GasUsed       uint64  `json:"gas_used"`
	Value         string  `json:"value"`
	ValueWei      string  `json:"value_wei"`
	Error         string  `json:"error"`
}

type ContractCreationInfo struct {
	Creator         string `json:"creator"`
	CreateType      string `json:"create_type"`
	NonceOnCreation uint64 `json:"nonce_on_creation"`
	Salt            string `json:"salt"`
	CodeHash        string `json:"code_hash"`
}

type TokenInitInfo struct {
	ContractAddr string `json:"contract_addr"`
	Name         string `json:"name"`
	Symble       string `json:"symble"`
	Decimals     string `json:"decimals"`
	TotalSupply  string `json:"total_supply"`
	OwnerBalance string `json:"owner_init_balance"`
	OwnerAddr    string `json:"owner_addr"`
	Type         string `json:"type"`
}

type ERC20Contract struct {
	ContractAddr []byte `json:"contract_addr"`
	ContractCode []byte `json:"contract_code"`
}

type BlockInnerData struct {
	BlockHash           string
	TxHashes            []string
	TxMap               map[string][]*InnerTxBasic
	ContractCreationMap map[string]map[string]*ContractCreationInfo
	ContractList        []*ERC20Contract
}

// InnerTx store all field of a innertx, you can change/add those field as you will.
type InnerTx struct {
	InnerTxBasic

	FromNonce       uint64
	Create2Salt     string
	Create2CodeHash string
}

func BuildInnerTxBasic(innerTxs []*InnerTx) []*InnerTxBasic {
	results := make([]*InnerTxBasic, 0, len(innerTxs))

	for _, innertx := range innerTxs {
		results = append(results, &innertx.InnerTxBasic)
	}

	return results
}

func BuildContractCreationInfos(innerTxs []*InnerTx) map[string]*ContractCreationInfo {
	results := make(map[string]*ContractCreationInfo)

	for _, tx := range innerTxs {
		if !tx.isCreateContract() {
			continue
		}

		createType := tx.CallType
		if tx.CallType == CosmosCallType && tx.Name == EvmCreateName {
			// TODO: define calltype constant
			createType = "create"
		}
		cci := &ContractCreationInfo{
			Creator:         tx.From,
			CreateType:      createType,
			NonceOnCreation: tx.FromNonce,
			Salt:            tx.Create2Salt,
			CodeHash:        tx.Create2CodeHash,
		}
		results[tx.To] = cci
	}

	return results
}

func (intx *InnerTx) isCreateContract() bool {
	// TODO: define calltype constant
	return intx.CallType == "create" ||
		intx.CallType == "create2" ||
		intx.Name == EvmCreateName
}

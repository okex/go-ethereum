package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type OKVerifier interface {
	Verify(stateDB StateDB, op OpCode, from, to common.Address, input []byte, value *big.Int) error
}

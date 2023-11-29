package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestAToIWithDec(t *testing.T) {
	testCases := []struct {
		name    string
		amt     string
		dec     *big.Int
		expect  *big.Int
		isError bool
	}{
		{
			name:    "normal1 ",
			amt:     "100",
			dec:     big.NewInt(1),
			expect:  big.NewInt(1000),
			isError: false,
		},
		{
			name:    "normal2 ",
			amt:     "100",
			dec:     big.NewInt(2),
			expect:  big.NewInt(10000),
			isError: false,
		},
		{
			name:    "normal3 ",
			amt:     "100.1",
			dec:     big.NewInt(2),
			expect:  big.NewInt(10010),
			isError: false,
		},
		{
			name:    "error1 ",
			amt:     ".100",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error2 ",
			amt:     "100.",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error3 ",
			amt:     "+100",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error4 ",
			amt:     "-100",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error5 ",
			amt:     "100e",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error6 ",
			amt:     "100E",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
		{
			name:    "error7 ",
			amt:     "100.0001",
			dec:     big.NewInt(2),
			expect:  big.NewInt(0),
			isError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			calldata, err := EncodeAToIWithDecInput(brcXABI, tc.amt, tc.dec)
			require.NoError(tt, err)
			result, err := atoiWithDec(calldata)
			if tc.isError {
				require.Error(tt, err)
				tt.Log(err)
			} else {
				expect, err := EncodeAToIWithDecOutput(brcXABI, tc.expect)
				require.NoError(tt, err)
				require.Equal(tt, expect, result)
			}
		})
	}
}

func EncodeAToIWithDecInput(abi abi.ABI, amt string, dec *big.Int) ([]byte, error) {
	method, ok := abi.Methods[AToIWithDec]
	if !ok {
		return nil, fmt.Errorf("method %s is not exist in abi", AToIWithDec)
	}
	buffer := make([]byte, 0)
	buffer = append(buffer, method.ID...)
	calldata, err := method.Inputs.Pack(amt, dec)
	if err != nil {
		return nil, err
	}
	buffer = append(buffer, calldata...)
	return buffer, nil
}

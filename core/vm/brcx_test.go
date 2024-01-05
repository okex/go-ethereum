package vm

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
		{
			name:    "error7 ",
			amt:     "100.000",
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

func EncodeIsSrc20TickAllCharValidIntput(abi abi.ABI, tick string) ([]byte, error) {
	method, ok := abi.Methods[IsSrc20TickAllCharValid]
	if !ok {
		return nil, fmt.Errorf("method %s is not exist in abi", IsSrc20TickAllCharValid)
	}
	buffer := make([]byte, 0)
	buffer = append(buffer, method.ID...)
	calldata, err := method.Inputs.Pack(tick)
	if err != nil {
		return nil, err
	}
	buffer = append(buffer, calldata...)
	return buffer, nil
}

func EncodeCountDecimalPlacesIntput(abi abi.ABI, num string) ([]byte, error) {
	method, ok := abi.Methods[CountDecimalPlaces]
	if !ok {
		return nil, fmt.Errorf("method %s is not exist in abi", CountDecimalPlaces)
	}
	buffer := make([]byte, 0)
	buffer = append(buffer, method.ID...)
	calldata, err := method.Inputs.Pack(num)
	if err != nil {
		return nil, err
	}
	buffer = append(buffer, calldata...)
	return buffer, nil
}

func EncodeFmtInscriptionIntput(abi abi.ABI, num string) ([]byte, error) {
	method, ok := abi.Methods[FmtInscription]
	if !ok {
		return nil, fmt.Errorf("method %s is not exist in abi", FmtInscription)
	}
	buffer := make([]byte, 0)
	buffer = append(buffer, method.ID...)
	calldata, err := method.Inputs.Pack(num)
	if err != nil {
		return nil, err
	}
	buffer = append(buffer, calldata...)
	return buffer, nil
}

func TestIsSrc20TickAllCharValidInput(t *testing.T) {
	testCases := []struct {
		name    string
		tick    string
		expect  bool
		isError bool
	}{
		{
			name:    "normal1",
			tick:    "🙂",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal2",
			tick:    "🙂APL",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal3",
			tick:    "APL",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal4",
			tick:    "APLLL",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal5",
			tick:    "🙂🙂🙂🙂🙂",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal6",
			tick:    "APL🙂🙂",
			expect:  true,
			isError: false,
		},
		{
			name:    "normal7",
			tick:    "A",
			expect:  true,
			isError: false,
		},
		{
			name:    "error1",
			tick:    "🙂🙂🙂🙂🙂🙂🙂",
			expect:  true,
			isError: true,
		},
		{
			name:    "error2",
			tick:    "AAAAAAA",
			expect:  true,
			isError: true,
		},
		{
			name:    "error3",
			tick:    "AA🙂🙂🙂🙂PP",
			expect:  true,
			isError: true,
		},
		{
			name:    "error4",
			tick:    "🙂🙂🙂🙂PPAA",
			expect:  true,
			isError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			calldata, err := EncodeIsSrc20TickAllCharValidIntput(brcXABI, tc.tick)
			require.NoError(tt, err)
			result, err := isSrc20TickAllCharValid(calldata)
			if tc.isError {
				require.Error(tt, err)
				tt.Log(err)
			} else {
				expect, err := EncodeIsSrc20TickAllCharValidOutput(brcXABI, tc.expect)
				require.NoError(tt, err)
				require.Equal(tt, expect, result)
			}
		})
	}
}

func TestCountDecimalPlacesInput(t *testing.T) {
	testCases := []struct {
		name    string
		num     string
		expect  uint8
		isError bool
	}{
		{
			name:    "normal1",
			num:     "1.1000",
			expect:  1,
			isError: false,
		},
		{
			name:    "normal2",
			num:     "1.1001",
			expect:  4,
			isError: false,
		},
		{
			name:    "normal3",
			num:     "1.0",
			expect:  0,
			isError: false,
		},
		{
			name:    "normal4",
			num:     "1.0000",
			expect:  0,
			isError: false,
		},
		{
			name:    "normal5",
			num:     "10000",
			expect:  0,
			isError: false,
		},
		{
			name:    "normal6",
			num:     ".10000",
			expect:  1,
			isError: false,
		},
		{
			name:    "normal7",
			num:     ".1",
			expect:  1,
			isError: false,
		},
		{
			name:    "normal8",
			num:     "100 00",
			expect:  0,
			isError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			calldata, err := EncodeCountDecimalPlacesIntput(brcXABI, tc.num)
			require.NoError(tt, err)
			result, err := countDecimalPlaces(calldata)
			if tc.isError {
				require.Error(tt, err)
				tt.Log(err)
			} else {
				expect, err := EncodeCountDecimalPlacesOutput(brcXABI, tc.expect)
				require.NoError(tt, err)
				require.Equal(tt, expect, result)
			}
		})
	}
}

func TestFmtInscriptionInput(t *testing.T) {
	testCases := []struct {
		name    string
		num     string
		expect  string
		isError bool
	}{
		{
			name:    "normal1",
			num:     "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"1.1000\"}",
			expect:  "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"1.1000\"}",
			isError: false,
		},
		{
			name:    "normal2",
			num:     "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"\"}",
			expect:  "",
			isError: true,
		},
		{
			name:    "normal3",
			num:     "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\": 5000}",
			expect:  "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"5000\"}",
			isError: false,
		},
		{
			name:    "normal3",
			num:     "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\": 5000.000}",
			expect:  "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"5000\"}",
			isError: false,
		},
		{
			name:    "normal3",
			num:     "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\": 5000.001}",
			expect:  "{\"p\":\"src-20\",\"op\":\"mint\",\"tick\":\"SATO\",\"amt\":\"5000.001\"}",
			isError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			calldata, err := EncodeFmtInscriptionIntput(brcXABI, tc.num)
			require.NoError(tt, err)
			result, err := fmtInscription(calldata)
			if tc.isError {
				require.Error(tt, err)
				tt.Log(err)
			} else {
				tmp := map[string]string{}
				json.Unmarshal([]byte(tc.expect), &tmp)
				ex, _ := json.Marshal(tmp)
				expect, err := EncodeFmtInscriptionOutput(brcXABI, string(ex))
				require.NoError(tt, err)
				require.Equal(tt, expect, result)
			}
		})
	}
}

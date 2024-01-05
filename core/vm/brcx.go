package vm

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	AToIWithDec             = "atoiWithDec"
	ToLower                 = "toLower"
	IsSrc20TickAllCharValid = "isSrc20TickAllCharValid"
	CountDecimalPlaces      = "countDecimalPlaces"
	FmtInscription          = "fmtInscription"
)

const Src20TickMaxLength = 5

var (
	brcXABI abi.ABI

	//go:embed brcx_abi.json
	preCompileJson []byte
)

func init() {
	var err error
	brcXABI, err = abi.JSON(bytes.NewReader(preCompileJson))
	if err != nil {
		panic(err)
	}
}

func GetMethodByID(abi abi.ABI, calldata []byte) (*abi.Method, error) {
	if len(calldata) < 4 {
		return nil, errors.New("the calldata length must more than 4")
	}
	sigdata := calldata[:4]
	argdata := calldata[4:]
	if len(argdata)%32 != 0 {
		return nil, fmt.Errorf("invalid call data; length should be a multiple of 32 bytes (was %d)", len(argdata))
	}

	return abi.MethodById(sigdata)
}

func IsMatchFunction(abi abi.ABI, methodName string, data []byte) bool {
	if len(data) < 4 {
		return false
	}
	method, ok := abi.Methods[methodName]
	if !ok {
		return false
	}
	if bytes.Equal(method.ID, data[:4]) {
		return true
	}
	return false
}

func DecodeInputParam(abi abi.ABI, methodName string, data []byte) ([]interface{}, error) {
	if len(data) <= 4 {
		return nil, fmt.Errorf("method %s data is nil", methodName)
	}
	method, ok := abi.Methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %s is not exist in abi", methodName)
	}
	return method.Inputs.Unpack(data[4:])
}

func DecodeAToIWithDecInput(abi abi.ABI, input []byte) (string, *big.Int, error) {
	if !IsMatchFunction(abi, AToIWithDec, input) {
		return "", nil, fmt.Errorf("decode precomplie call : input sginature is not %s", AToIWithDec)
	}
	unpacked, err := DecodeInputParam(abi, AToIWithDec, input)
	if err != nil {
		return "", nil, fmt.Errorf("decode precomplie call : input unpack err :  %s", err)
	}
	if len(unpacked) != 2 {
		return "", nil, fmt.Errorf("decode precomplie call to wasm input unpack err :  unpack data len expect 2 but got %v", len(unpacked))
	}
	amt, ok := unpacked[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("decode precomplie call : input unpack err : wasmAddr is not type of string")
	}
	dec, ok := unpacked[1].(*big.Int)
	if !ok {
		return "", nil, fmt.Errorf("decode precomplie call : input unpack err : calldata is not type of string")
	}
	return amt, dec, nil
}

func EncodeAToIWithDecOutput(abi abi.ABI, result *big.Int) ([]byte, error) {
	method, ok := abi.Methods[AToIWithDec]
	if !ok {
		return make([]byte, 0), fmt.Errorf("can not found method for abi")
	}

	return method.Outputs.PackValues([]interface{}{result})
}

func DecodeToLowerInput(abi abi.ABI, input []byte) (string, error) {
	if !IsMatchFunction(abi, ToLower, input) {
		return "", fmt.Errorf("decode precomplie call : input sginature is not %s", ToLower)
	}
	unpacked, err := DecodeInputParam(abi, ToLower, input)
	if err != nil {
		return "", fmt.Errorf("decode precomplie call : input unpack err :  %s", err)
	}
	if len(unpacked) != 1 {
		return "", fmt.Errorf("decode precomplie call to wasm input unpack err :  unpack data len expect 2 but got %v", len(unpacked))
	}
	str, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("decode precomplie call : input unpack err : wasmAddr is not type of string")
	}

	return str, nil
}

func DecodeCountDecimalPlacesInput(abi abi.ABI, input []byte) (string, error) {
	if !IsMatchFunction(abi, CountDecimalPlaces, input) {
		return "", fmt.Errorf("decode precomplie call : input sginature is not %s", CountDecimalPlaces)
	}

	unpacked, err := DecodeInputParam(abi, CountDecimalPlaces, input)
	if err != nil {
		return "", fmt.Errorf("decode precomplie call : input unpack err :  %s", err)
	}

	if len(unpacked) != 1 {
		return "", fmt.Errorf("decode precomplie call to CountDecimalPlaces input unpack err :  unpack data len expect 1 but got %v", len(unpacked))
	}
	str, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("decode precomplie call : input unpack err : num is not type of string")
	}

	return str, nil
}

func EncodeCountDecimalPlacesOutput(abi abi.ABI, dec uint8) ([]byte, error) {
	method, ok := abi.Methods[CountDecimalPlaces]
	if !ok {
		return make([]byte, 0), fmt.Errorf("can not found method for abi")
	}

	return method.Outputs.PackValues([]interface{}{dec})
}

func EncodeToLowerOutput(abi abi.ABI, result string) ([]byte, error) {
	method, ok := abi.Methods[ToLower]
	if !ok {
		return make([]byte, 0), fmt.Errorf("can not found method for abi")
	}

	return method.Outputs.PackValues([]interface{}{result})
}

func EncodeIsSrc20TickAllCharValidOutput(abi abi.ABI, result bool) ([]byte, error) {
	method, ok := abi.Methods[IsSrc20TickAllCharValid]
	if !ok {
		return make([]byte, 0), fmt.Errorf("can not found method for abi")
	}

	return method.Outputs.PackValues([]interface{}{result})
}

func DecodeIsSrc20TickAllCharValidInput(abi abi.ABI, input []byte) (string, error) {
	if !IsMatchFunction(abi, IsSrc20TickAllCharValid, input) {
		return "", fmt.Errorf("decode precomplie call : input sginature is not %s", IsSrc20TickAllCharValid)
	}
	unpacked, err := DecodeInputParam(abi, IsSrc20TickAllCharValid, input)
	if err != nil {
		return "", fmt.Errorf("decode precomplie call : input unpack err :  %s", err)
	}
	if len(unpacked) != 1 {
		return "", fmt.Errorf("decode precomplie call unpack err :  unpack data len expect 1 but got %v", len(unpacked))
	}

	tick, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("decode precomplie call : input unpack err : num is not type of string")
	}
	return tick, nil
}

func DecodeFmtInscriptionInput(abi abi.ABI, input []byte) (string, error) {
	if !IsMatchFunction(abi, FmtInscription, input) {
		return "", fmt.Errorf("decode precomplie call : input sginature is not %s", FmtInscription)
	}

	unpacked, err := DecodeInputParam(abi, FmtInscription, input)
	if err != nil {
		return "", fmt.Errorf("decode precomplie call : input unpack err :  %s", err)
	}

	if len(unpacked) != 1 {
		return "", fmt.Errorf("decode precomplie call to FmtInscription input unpack err :  unpack data len expect 1 but got %v", len(unpacked))
	}
	str, ok := unpacked[0].(string)
	if !ok {
		return "", fmt.Errorf("decode precomplie call : input unpack err : num is not type of string")
	}

	return str, nil
}

func EncodeFmtInscriptionOutput(abi abi.ABI, res string) ([]byte, error) {
	method, ok := abi.Methods[FmtInscription]
	if !ok {
		return make([]byte, 0), fmt.Errorf("can not found method for abi")
	}

	return method.Outputs.PackValues([]interface{}{res})
}

// atoiwithdec convert to integer from str with decimal
type brcXContract struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *brcXContract) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *brcXContract) Run(input []byte) ([]byte, error) {
	method, err := GetMethodByID(brcXABI, input)
	if err != nil {
		return make([]byte, 0), err
	}

	switch method.Name {
	case AToIWithDec:
		return atoiWithDec(input)
	case ToLower:
		return toLower(input)
	case IsSrc20TickAllCharValid:
		return isSrc20TickAllCharValid(input)
	case CountDecimalPlaces:
		return countDecimalPlaces(input)
	case FmtInscription:
		return fmtInscription(input)
	default:
		return make([]byte, 0), fmt.Errorf("unsupport method: %s", method.Name)

	}
}

func atoiWithDec(calldata []byte) ([]byte, error) {
	amt, dec, err := DecodeAToIWithDecInput(brcXABI, calldata)
	if err != nil {
		return make([]byte, 0), err
	}
	if dec.Int64() > 18 {
		return make([]byte, 0), fmt.Errorf("decimals %d too large", dec.Int64())
	}
	if strings.HasPrefix(amt, ".") || strings.HasSuffix(amt, ".") || strings.Contains(amt, "e") || strings.Contains(amt, "E") || strings.Contains(amt, "+") || strings.Contains(amt, "-") {
		return make([]byte, 0), fmt.Errorf("invalid number: %s", amt)
	}
	amount, err := decimal.NewFromString(amt)
	if err != nil {
		return make([]byte, 0), err
	}
	if math.Abs(float64(amount.Exponent())) > float64(dec.Int64()) {
		return make([]byte, 0), fmt.Errorf("amount overflow: %s", amt)
	}

	resultDec := amount.Shift(int32(dec.Int64()))
	if resultDec.IsPositive() && resultDec.IsInteger() {
		return EncodeAToIWithDecOutput(brcXABI, resultDec.BigInt())
	} else {
		return make([]byte, 0), fmt.Errorf("invalid number: must be postive and integer %s", amt)
	}
}

func toLower(calldata []byte) ([]byte, error) {
	str, err := DecodeToLowerInput(brcXABI, calldata)
	if err != nil {
		return make([]byte, 0), err
	}

	return EncodeToLowerOutput(brcXABI, strings.ToLower(str))
}

func isSrc20TickAllCharValid(calldata []byte) ([]byte, error) {
	tick, err := DecodeIsSrc20TickAllCharValidInput(brcXABI, calldata)
	if err != nil {
		return make([]byte, 0), err
	}

	res, err := src20tickAllCharValid(tick)
	if err != nil {
		return make([]byte, 0), err
	}

	return EncodeIsSrc20TickAllCharValidOutput(brcXABI, res)
}

func countDecimalPlaces(callData []byte) ([]byte, error) {
	num, err := DecodeCountDecimalPlacesInput(brcXABI, callData)
	if err != nil {
		return make([]byte, 0), err
	}

	dec, err := countDec(num)
	if err != nil {
		return make([]byte, 0), err
	}

	return EncodeCountDecimalPlacesOutput(brcXABI, dec)
}

func countDec(strNumber string) (uint8, error) {
	if !validNumFmt(strNumber) {
		return 0, fmt.Errorf("invalid fmt of num")
	}
	dotIndex := strings.Index(strNumber, ".")
	if dotIndex == -1 {
		return 0, nil
	}
	decimalPart := strNumber[dotIndex+1:]
	res := len(decimalPart)
	for i := len(decimalPart) - 1; i >= 0; i-- {
		if decimalPart[i] == '0' {
			res--
		} else {
			break
		}
	}
	return uint8(res), nil
}

func fmtInscription(callData []byte) ([]byte, error) {
	inscription, err := DecodeFmtInscriptionInput(brcXABI, callData)
	if err != nil {
		return make([]byte, 0), err
	}

	res, err := fmtInscriptionCore(inscription)
	if err != nil {
		return make([]byte, 0), err
	}

	return EncodeFmtInscriptionOutput(brcXABI, res)
}

func fmtInscriptionCore(inscription string) (string, error) {
	mapInscription := make(map[string]interface{})
	err := json.Unmarshal([]byte(inscription), &mapInscription)
	if err != nil {
		return "", nil
	}
	if amt, ok := mapInscription["amt"]; ok {
		amtStr, err := validNumValue(amt)
		if err != nil {
			return "", err
		}
		mapInscription["amt"] = amtStr
	}
	if lim, ok := mapInscription["lim"]; ok {
		limStr, err := validNumValue(lim)
		if err != nil {
			return "", err
		}
		mapInscription["lim"] = limStr
	}
	if max, ok := mapInscription["max"]; ok {
		maxStr, err := validNumValue(max)
		if err != nil {
			return "", err
		}
		mapInscription["max"] = maxStr
	}

	res, err := json.Marshal(mapInscription)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func validNumValue(num interface{}) (string, error) {
	res := ""
	switch num.(type) {
	case string:
		res = num.(string)
	case float64:
		res = strconv.FormatFloat(num.(float64), 'f', -1, 64)
	default:
		return "", fmt.Errorf("invalid value of num")
	}
	if !validNumFmt(res) {
		return "", fmt.Errorf("invalid fmt of num")
	}
	return res, nil
}

func validNumFmt(numStr string) bool {
	re := regexp.MustCompile(`^[0-9.]+$`)
	return re.MatchString(numStr)
}

func src20tickAllCharValid(data string) (bool, error) {
	input := []rune(data)
	if len(input) == 0 || len(input) > Src20TickMaxLength {
		return false, fmt.Errorf("num length %d invalid", len(data))
	}

	vm := goja.New()
	new(require.Registry).Enable(vm)
	console.Enable(vm)

	script := `
		function tickAllCharValid(data) {
			let reEmoji = /\p{Emoji_Modifier_Base}\p{Emoji_Modifier}?|\p{Emoji_Presentation}|\p{Emoji}\uFE0F/gu;
			let reCommon = /[\w~!@#$%^&*()_+=<>?]/;
			reCommon.lastIndex = 0;
			reEmoji.lastIndex = 0;
			for (let i = 0; i < data.length; i++) {
        		var char = data[i];
				if (reEmoji.test(char) || reCommon.test(char)) {
				} else {
					return false;
				}
			}
			return true;
		}
	`
	prog, err := goja.Compile("", script, true)
	if err != nil {
		return false, fmt.Errorf("Error compiling the script %v ", err.Error())
	}
	_, err = vm.RunProgram(prog)

	var myJSFunc goja.Callable
	err = vm.ExportTo(vm.Get("tickAllCharValid"), &myJSFunc)
	if err != nil {
		fmt.Printf("Error exporting the function %v", err)
		return false, fmt.Errorf("Error compiling the script %v ", err.Error())
	}

	res, err := myJSFunc(goja.Undefined(), vm.ToValue(input))
	if err != nil {
		return false, fmt.Errorf("error calling function %s", err.Error())
	}
	return res.ToBoolean(), nil
}

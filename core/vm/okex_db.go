package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

type OKDB interface {
	// Get fetches the value of the given key, or nil if it does not exist.
	// CONTRACT: key, value readonly []byte
	Get([]byte) ([]byte, error)

	// Set sets the value for the given key, replacing it if it already exists.
	// CONTRACT: key, value readonly []byte
	Set([]byte, []byte) error

	// Close closes the database connection.
	Close() error
}

type DBCreator func(name string, dir string) (OKDB, error)

var (
	txDB    OKDB
	blockDB OKDB
	tokenDB OKDB

	contractCreationDB OKDB
)

func InitDB(dir string, creator DBCreator) error {
	if creator == nil {
		return fmt.Errorf("createor is nil %v", dir)
	}

	var dbDir string
	if dir != "" {
		dbDir = filepath.Join(dir, "okdb")
	} else {
		dbDir = filepath.Join("okdb")
	}
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		err := os.MkdirAll(dbDir, 0700)
		if err != nil {
			return fmt.Errorf("could not create directory %v: %w", dir, err)
		}
	}

	txDB = newDB("InnerTxDB", creator, dbDir)
	blockDB = newDB("InnerBlockDB", creator, dbDir)
	tokenDB = newDB("TokenDB", creator, dbDir)
	contractCreationDB = newDB("ContractCreationDB", creator, dbDir)

	return nil
}

func newDB(name string, creator DBCreator, dir string) OKDB {
	db, err := creator(name, dir)
	if err != nil {
		panic(fmt.Sprintf("Error initializing DB: %v", err))
	}
	return db
}

func CloseDB() (errors []error) {
	if err := txDB.Close(); err != nil {
		errors = append(errors, err)
	}

	if err := blockDB.Close(); err != nil {
		errors = append(errors, err)
	}

	if err := tokenDB.Close(); err != nil {
		errors = append(errors, err)
	}

	if err := contractCreationDB.Close(); err != nil {
		errors = append(errors, err)
	}

	log.Info("Close innerTx DB")
	return errors
}

func ReadToken(key []byte) []byte {
	rtn, _ := tokenDB.Get(key)
	return rtn
}

func WriteTx(hash string, ix []*InnerTxBasic) error {
	row, _ := rlp.EncodeToBytes(ix)
	err := txDB.Set([]byte(hash), row)
	if err != nil {
		log.Info("WriteTx error:" + hash)
		return err
	}
	return nil
}

func WriteContractCreationInfo(addr string, info *ContractCreationInfo) error {
	contractAddr := common.HexToAddress(addr).String()

	row, _ := rlp.EncodeToBytes(info)
	err := contractCreationDB.Set([]byte(strings.ToLower(contractAddr)), row)
	if err != nil {
		log.Info("WriteContractCreationInfo error:" + contractAddr)
		return err
	}
	return nil
}

func WriteBlockDB(blockhash string, hash []string) error {
	if len(blockhash) != 0 && len(hash) != 0 {
		row, _ := rlp.EncodeToBytes(hash)
		err := blockDB.Set([]byte(blockhash), row)
		if err != nil {
			log.Info("WriteBlock err:" + blockhash)
			return err
		}
	}
	return nil
}

func WriteToken(key []byte, value []byte) error {
	err := tokenDB.Set(key, value)
	if err != nil {
		log.Info("WriteToken error:" + string(key))
		return err
	}
	return nil
}

func GetFromDB(hash string) []InnerTxBasic {
	result, err := txDB.Get([]byte(hash))
	if err != nil {
		return nil
	}

	innerTxs := make([]InnerTxBasic, 0)
	rlp.DecodeBytes(result, &innerTxs)
	return innerTxs
}

func GetBlockDB(blockHash string) []string {
	result, err := blockDB.Get([]byte(blockHash))
	if err != nil {
		return nil
	}

	var innerTxs []string
	rlp.DecodeBytes(result, &innerTxs)
	return innerTxs
}

func GetContractCreationDB(addr common.Address) (ContractCreationInfo, error) {
	contractAddr := addr.String()

	result, err := contractCreationDB.Get([]byte(strings.ToLower(contractAddr)))
	if err != nil {
		return ContractCreationInfo{}, err
	}
	if len(result) == 0 {
		return ContractCreationInfo{}, fmt.Errorf("'%s' not exist in creattion info", contractAddr)
	}

	var info ContractCreationInfo
	err = rlp.DecodeBytes(result, &info)
	return info, err
}

// We defensively turn nil keys or values into []byte{} for
// most operations.
func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}

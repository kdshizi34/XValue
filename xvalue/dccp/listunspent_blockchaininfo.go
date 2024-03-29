package dccp

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/xvalue/go-xvalue/log"
)

func LockoutIsConfirmed(addr string,txhash string) (bool,error) {
	resstr := GetUTXO_BlockChainInfo(addr)
	utxoLsRes, err := parseUnspent(resstr)
	log.Debug("=============LockoutIsConfirmed,","utxoLsRes",utxoLsRes)
	if err != nil {
	    log.Debug("======LockoutIsConfirmed,return err.==========")
		return false, err
	}
	
	for _, utxo := range utxoLsRes.Unspent_outputs {
	    if strings.EqualFold(utxo.Tx_hash_big_endian,txhash) && utxo.Confirmations >= BTC_BLOCK_CONFIRMS {
		return true,nil
	    }
	}

	return false,errors.New("it is not confirmed.")
}

func listUnspent_blockchaininfo(addr string) ([]btcjson.ListUnspentResult, error) {
	resstr := GetUTXO_BlockChainInfo(addr)
	utxoLsRes, err := parseUnspent(resstr)
	log.Debug("=============listUnspent_blockchaininfo,","utxoLsRes",utxoLsRes)
	if err != nil {
	    log.Debug("======listUnspent_blockchaininfo,return err.==========")
		return nil, err
	}
	//var list []btcjson.ListUnspentResult
	var list sortableLURSlice
	for _, utxo := range utxoLsRes.Unspent_outputs {
		res := btcjson.ListUnspentResult{
			TxID: utxo.Tx_hash_big_endian,
			Vout: uint32(utxo.Tx_output_n),
			Address: addr,
			ScriptPubKey: utxo.Script,
			//RedeemScript:
			Amount: utxo.Value/1e8,
			Confirmations: utxo.Confirmations,
			Spendable: true,
		}
		list = append(list, res)
	}
	sort.Sort(list)
	return list, nil
}

func parseUnspent(resstr string) (UtxoLsRes, error) {
	resstr = strings.Replace(resstr, " ", "", -1)
	resstr = strings.Replace(resstr, "\n", "", -1)

	last_index := len(resstr)-1
	for last_index > 0 {
		if resstr[last_index] != '}' {
			last_index --
		} else {
			break
		}
	}
	res := &UtxoLsRes{}
	err := json.Unmarshal([]byte(resstr)[:last_index+1], res)
	return *res, err
}

type UtxoLsRes struct {
	Unspent_outputs []UtxoRes
}

type UtxoRes struct {
	Tx_hash_big_endian	string
	Script			string
	Tx_output_n		uint32
	Value			float64
	Confirmations		int64
}

func GetUTXO_BlockChainInfo (addr string) string {
	addrReceivedUrl := "https://testnet.blockchain.info/unspent?active=" + addr
	blockchaininfores := loginPre1("GET",addrReceivedUrl)
	log.Debug("=============GetUTXO_BlockChainInfo,","blockchaininfores",blockchaininfores)
	return blockchaininfores
}


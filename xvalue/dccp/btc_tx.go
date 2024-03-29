// Copyright 2018 The xvalue-dccp 


package dccp

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"

	"github.com/xvalue/go-xvalue/crypto"
	"github.com/xvalue/go-xvalue/log"
	"strconv"
	"time"
)

//func main() {}

type ListUTXOs struct {
    RESULT []ListUTXO
    ERROR error 
    ID int
}
type ListUTXO struct {
    TXID string
    VOUT uint32
    ADDRESS string
    ACCOUNT string
    SCRIPTPUBKEY string
    AMOUNT float64
    CONFIRMATIONS int64
    SPENDABLE     bool
    SOLVABLE bool
    SAFE bool
}

var opts = struct {
	Dccpaddr              string  // from地址
	Toaddr                string  // to地址
	ChangeAddress    string  // 找零地址
	Value                 float64  // 发送value
        RequiredConfirmations int64
	FeeRate               *btcutil.Amount  // 费用率
}{
	Dccpaddr: "",
	Toaddr: "",
	ChangeAddress: "",
	Value: 0,
	RequiredConfirmations: BTC_BLOCK_CONFIRMS,
	FeeRate: &feeRate,
}
var feeRate, _ = btcutil.NewAmount(BTC_DEFAULT_FEE)

// 构建和发送一笔比特币交易
// dccpaddr: dccp地址, toAddr: 接收转账的地址
// changeAddress: 找零地址
// requiredConfirmations 需要的确认区块数, 默认是6
// feeRateBtc: 费用率, 单位是比特币
func Btc_createTransaction(msgprex string,dccpaddr string, toAddr string, changeAddress string, value float64, requiredConfirmations uint32, feeRateBtc float64,ch chan interface{}) string {
	opts.Dccpaddr = dccpaddr
	opts.Toaddr = toAddr
	opts.ChangeAddress = changeAddress
	opts.Value = value
	if requiredConfirmations >= 1 {
		opts.RequiredConfirmations = int64(requiredConfirmations)
	}
	var feeRate, _ = btcutil.NewAmount(feeRateBtc)
	opts.FeeRate = &feeRate

	txhash,err := btc_createTransaction(msgprex,dccpaddr,ch)
	if err != nil {
		log.Debug("","create btc tx error", err)
		return ""
	}

	log.Debug("==========Btc_createTransaction,return tx hash.=============")
	return txhash
}

func ImportAddrToWallet(dccpaddr string,ch chan bool) {
	c, err := NewClient(SERVER_HOST, SERVER_PORT, USER, PASSWD, USESSL)
	if err != nil {
	    log.Debug("============ImportAddrToWallet,new client fail.===========")
	    ch<- false
	    return
	}
	
	jsoncmd := "{\"jsonrpc\":\"1.0\",\"method\":\"importaddress\",\"params\":[\"" + dccpaddr + "\"" + "," + "\"watch-only test\",true]" + "," + "\"id\":\"1\"}";
	retdata,err := c.Send(jsoncmd)
	time.Sleep(time.Duration(180)*time.Second) //1000 == 1s
	if err != nil {
	    log.Debug("============ImportAddrToWallet","err",err.Error(),"","===========")
	    ch<- false
	    return
	}

	log.Debug("============ImportAddrToWallet","get return data",retdata,"","===========")
	ch<- true 
}

func listUnspentWithWallet(dccpaddr string) ([]btcjson.ListUnspentResult, error) {
	rch := make (chan bool,1)
	go ImportAddrToWallet(dccpaddr,rch)
	_,cherr := GetChannelValue(360,rch)
	if cherr != nil {
	    log.Debug("============listUnspentWithWallet,","get error",cherr.Error(),"","==============")
	    return nil,cherr 
	}

	c, err := NewClient(SERVER_HOST, SERVER_PORT, USER, PASSWD, USESSL)
	if err != nil {
	    log.Debug("============listUnspentWithWallet,new client fail.===========")
	    return nil,err
	}
	
	jsoncmd := "{\"jsonrpc\":\"1.0\",\"method\":\"listunspent\",\"params\":[1, 9999999,[\"" + dccpaddr + "\"]],\"id\":\"1\"}";
	retdata,err := c.Send(jsoncmd)
	if err != nil {
	    log.Debug("============listUnspentWithWallet","err",err.Error(),"","===========")
	    return nil,err
	}
	log.Debug("============listUnspentWithWallet","ret data",retdata,"","===========")
	var uos ListUTXOs
	json.Unmarshal([]byte(retdata), &uos)
	/*if ok != nil {
	    log.Debug("============listUnspentWithWallet,no get utxos.===========")
	    return nil,errors.New("no get utxos.")
	}*/

	log.Debug("============listUnspentWithWallet","uos",uos,"","===========")
	var list sortableLURSlice
	for _, utxo := range uos.RESULT {
		res := btcjson.ListUnspentResult{
			TxID: utxo.TXID,
			Vout: utxo.VOUT,
			Address: utxo.ADDRESS,
			Account:utxo.ACCOUNT,
			ScriptPubKey: utxo.SCRIPTPUBKEY,
			RedeemScript:"",
			Amount: utxo.AMOUNT,
			Confirmations: utxo.CONFIRMATIONS,
			Spendable: true,
		}
		list = append(list, res)
	}
	sort.Sort(list)
	return list,nil
}

func GetTxOutsAmount(lockoutto string,value float64) *big.Int {
        if lockoutto == "" || value <= 0 {
	    return nil 
	}

	// 设置交易输出
	var txOuts []*wire.TxOut
	cfg := chaincfg.MainNetParams
	toAddr, _ := btcutil.DecodeAddress(lockoutto, &cfg)
	pkscript, _ := txscript.PayToAddrScript(toAddr)
	txOut := wire.NewTxOut(int64(value),pkscript)
	txOuts = append(txOuts,txOut)
	for _, txo := range txOuts {
		log.Debug("","txo",txo)
		log.Debug("","txo value",txo.Value)
	}
	
	targetAmount := SumOutputValues(txOuts)
	estimatedSize := EstimateVirtualSize(0, 1, 0, txOuts, true)
	targetFee := txrules.FeeForSerializeSize(*opts.FeeRate, estimatedSize)
	 input := (targetAmount+targetFee).ToBTC()*100000000 //??BTC
	data := strconv.FormatFloat(input, 'f', 0, 64)
	//data := fmt.Sprintf("%v",input)
	 log.Debug("==========GetTxOutsAmount,","data",data,"","=============")
	 //ins := strconv.FormatFloat(input, 'f', -1, 64)
	 total,_ := new(big.Int).SetString(data,10)
	return total
}

func ChooseDccpAddrForLockoutByValue(dccpaddr string,lockoutto string,value float64) bool {

        if dccpaddr == "" || lockoutto == "" || value <= 0 {
	    return false
	}

	log.Debug("==========ChooseDccpAddrForLockoutByValue,","dccpaddr",dccpaddr,"lockoutto",lockoutto,"value",value,"","================")
	//unspentOutputs, err := listUnspent_blockchaininfo(dccpaddr)
	unspentOutputs, err := listUnspent(dccpaddr)
	//unspentOutputs, err := listUnspentWithWallet(dccpaddr)
	log.Debug("=========ChooseDccpAddrForLockoutByValue,","unspentOutputs",unspentOutputs,"","================")
	if err != nil {
		return false
	}
	//////////bug/////
	if len(unspentOutputs) == 0 && err == nil {
	    time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
	    unspentOutputs, err = listUnspent_blockchaininfo(dccpaddr) //try again
	    if len(unspentOutputs) == 0 && err == nil {
		time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
		unspentOutputs, err = listUnspent(dccpaddr) //try again
	    }
	}
	/////////////////

	sourceOutputs := make(map[string][]btcjson.ListUnspentResult)
	
	for _, unspentOutput := range unspentOutputs {
		if !unspentOutput.Spendable {
		    log.Debug("=========ChooseDccpAddrForLockoutByValue,un-spendable=======")
			continue
		}
		if unspentOutput.Confirmations < opts.RequiredConfirmations {
			log.Debug("=========ChooseDccpAddrForLockoutByValue,< confirms=======")
			continue
		}
		sourceAddressOutputs := sourceOutputs[unspentOutput.Address]
		sourceOutputs[unspentOutput.Address] = append(sourceAddressOutputs, unspentOutput)
	}
	log.Debug("","sourceOutputs",sourceOutputs)

	// 设置交易输出
	var txOuts []*wire.TxOut
	cfg := chaincfg.MainNetParams
	toAddr, _ := btcutil.DecodeAddress(lockoutto, &cfg)
	pkscript, _ := txscript.PayToAddrScript(toAddr)
	txOut := wire.NewTxOut(int64(value),pkscript)
	txOuts = append(txOuts,txOut)
	for _, txo := range txOuts {
		log.Debug("","txo",txo)
		log.Debug("","txo value",txo.Value)
	}

	/*var numErrors int
	var reportError = func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
		os.Stderr.Write([]byte{'\n'})
		numErrors++
	}*/

	for _, previousOutputs := range sourceOutputs {

		targetAmount := SumOutputValues(txOuts)
		estimatedSize := EstimateVirtualSize(0, 1, 0, txOuts, true)
		targetFee := txrules.FeeForSerializeSize(*opts.FeeRate, estimatedSize)

		//设置输入
		var inputSource txauthor.InputSource
		for i, _ := range previousOutputs {
			inputSource = makeInputSource(previousOutputs[:i+1])
			inputAmount, _, _, _, err := inputSource(targetAmount + targetFee)
			if err != nil {
				return false
			}
			if inputAmount < targetAmount+targetFee {
				continue
			} else {
				return true
			}
		}
	 }

	 return false
}

func GetDccpAddrBalanceForLockout(dccpaddr string,lockoutto string,value float64) (*big.Int,bool) {

        zero,_ := new(big.Int).SetString("0",10)
        if dccpaddr == "" || lockoutto == "" || value <= 0 {
	    return zero,false
	}

	log.Debug("==========GetDccpAddrBalanceForLockout,","dccpaddr",dccpaddr,"lockoutto",lockoutto,"value",value,"","================")
	//unspentOutputs, err := listUnspent_blockchaininfo(dccpaddr)
	unspentOutputs, err := listUnspent(dccpaddr)
	//////////bug/////
	if len(unspentOutputs) == 0 && err == nil {
	    time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
	    unspentOutputs, err = listUnspent_blockchaininfo(dccpaddr) //try again
	    if len(unspentOutputs) == 0 && err == nil {
		time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
		unspentOutputs, err = listUnspent(dccpaddr) //try again
	    }
	}
	/////////////////
	//unspentOutputs, err := listUnspentWithWallet(dccpaddr)
	log.Debug("=========GetDccpAddrBalanceForLockout,","unspentOutputs",unspentOutputs,"","================")
	if err != nil {
		return zero,false
	}
	sourceOutputs := make(map[string][]btcjson.ListUnspentResult)
	
	for _, unspentOutput := range unspentOutputs {
		if !unspentOutput.Spendable {
		    log.Debug("=========GetDccpAddrBalanceForLockout,un-spendable=======")
			continue
		}
		if unspentOutput.Confirmations < opts.RequiredConfirmations {
			log.Debug("=========GetDccpAddrBalanceForLockout,< confirms=======")
			continue
		}
		sourceAddressOutputs := sourceOutputs[unspentOutput.Address]
		sourceOutputs[unspentOutput.Address] = append(sourceAddressOutputs, unspentOutput)
	}
	log.Debug("","sourceOutputs",sourceOutputs)

	// 设置交易输出
	var txOuts []*wire.TxOut
	cfg := chaincfg.MainNetParams
	toAddr, _ := btcutil.DecodeAddress(lockoutto, &cfg)
	pkscript, _ := txscript.PayToAddrScript(toAddr)
	txOut := wire.NewTxOut(int64(value),pkscript)
	txOuts = append(txOuts,txOut)
	for _, txo := range txOuts {
		log.Debug("","txo",txo)
		log.Debug("","txo value",txo.Value)
	}

	var inputAmount btcutil.Amount

	for _, previousOutputs := range sourceOutputs {

		targetAmount := SumOutputValues(txOuts)
		estimatedSize := EstimateVirtualSize(0, 1, 0, txOuts, true)
		targetFee := txrules.FeeForSerializeSize(*opts.FeeRate, estimatedSize)

		//设置输入
		var inputSource txauthor.InputSource
		for i, _ := range previousOutputs {
			inputSource = makeInputSource(previousOutputs[:i+1])
			inputAmount,_, _, _, err = inputSource(targetAmount + targetFee)
			if err != nil {
				return zero,false
			}
		}
	 }

	 log.Debug("==========GetDccpAddrBalanceForLockout,","inputAmount",inputAmount,"","=============")
	 input := inputAmount.ToBTC()*100000000 //??BTC
	data := strconv.FormatFloat(input, 'f', 0, 64)
	//data := fmt.Sprintf("%v",input)
	 log.Debug("==========GetDccpAddrBalanceForLockout,","data",data,"","=============")
	 //ins := strconv.FormatFloat(input, 'f', -1, 64)
	 total,_ := new(big.Int).SetString(data,10)
	 return total,true
}

func GetBTCTxFee(lockoutto string,value float64) (*big.Int,error) {
    if lockoutto == "" || value <= 0 {
	return nil,errors.New("get fee error.")
    }

    // 设置交易输出
    var txOuts []*wire.TxOut
    cfg := chaincfg.MainNetParams
    toAddr, _ := btcutil.DecodeAddress(lockoutto, &cfg)
    pkscript, _ := txscript.PayToAddrScript(toAddr)
    txOut := wire.NewTxOut(int64(value),pkscript)
    txOuts = append(txOuts,txOut)
    for _, txo := range txOuts {
	    log.Debug("","txo",txo)
	    log.Debug("","txo value",txo.Value)
    }

    estimatedSize := EstimateVirtualSize(0, 1, 0, txOuts, true)
    targetFee := txrules.FeeForSerializeSize(*opts.FeeRate, estimatedSize)
     log.Debug("==========GetBTCTxFee,","targetFee",targetFee,"","=============")
     input := targetFee.ToBTC()*100000000 //??BTC
    data := strconv.FormatFloat(input, 'f', 0, 64)
    //data := fmt.Sprintf("%v",input)
     log.Debug("==========GetBTCTxFee,","data",data,"","=============")
     total,_ := new(big.Int).SetString(data,10)
    return total,nil
}

func btc_createTransaction(msgprex string,dccpaddr string,ch chan interface{}) (string,error) {
	//fmt.Printf("\n============ start ============\n\n\n")

	// Fetch all unspent outputs, ignore those not from the source
	// account, and group by their change address.  Each grouping of
	// outputs will be used as inputs for a single transaction sending to a
	// new change account address.
	unspentOutputs, err := listUnspent_blockchaininfo(opts.Dccpaddr)
	//unspentOutputs, err := listUnspentWithWallet(opts.Dccpaddr)

	if err != nil {
		return "",errContext(err, "failed to fetch unspent outputs")
	}
	//////////bug/////
	if len(unspentOutputs) == 0 && err == nil {
	    time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
	    unspentOutputs, err = listUnspent(dccpaddr) //try again
	    if len(unspentOutputs) == 0 && err == nil {
		time.Sleep(time.Duration(2)*time.Second) //1000 == 1s
		unspentOutputs, err = listUnspent_blockchaininfo(dccpaddr) //try again
	    }
	}
	/////////////////
	sourceOutputs := make(map[string][]btcjson.ListUnspentResult)
	
	for _, unspentOutput := range unspentOutputs {
		if !unspentOutput.Spendable {
			continue
		}
		if unspentOutput.Confirmations < opts.RequiredConfirmations {
			continue
		}
		sourceAddressOutputs := sourceOutputs[unspentOutput.Address]
		sourceOutputs[unspentOutput.Address] = append(sourceAddressOutputs, unspentOutput)
	}
	log.Debug("","sourceOutputs",sourceOutputs)

	// 设置交易输出
	var txOuts []*wire.TxOut
	cfg := chaincfg.MainNetParams
	toAddr, _ := btcutil.DecodeAddress(opts.Toaddr, &cfg)
	pkscript, _ := txscript.PayToAddrScript(toAddr)
	txOut := wire.NewTxOut(int64(opts.Value),pkscript)
	txOuts = append(txOuts,txOut)
	for _, txo := range txOuts {
		log.Debug("","txo",txo)
		log.Debug("","txo value",txo.Value)
	}

	var numErrors int
	var reportError = func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
		os.Stderr.Write([]byte{'\n'})
		numErrors++
	}

	//对每一个地址，用它的utxo作为输入，构建和发送一笔交易
	for _, previousOutputs := range sourceOutputs {

	        log.Debug("=========start new tx==========")
		targetAmount := SumOutputValues(txOuts)
		estimatedSize := EstimateVirtualSize(0, 1, 0, txOuts, true)
		targetFee := txrules.FeeForSerializeSize(*opts.FeeRate, estimatedSize)

		//设置输入
		var inputSource txauthor.InputSource
		for i, _ := range previousOutputs {
			inputSource = makeInputSource(previousOutputs[:i+1])
			inputAmount, _, _, _, err := inputSource(targetAmount + targetFee)
			if err != nil {
				return "",err
			}
			if inputAmount < targetAmount+targetFee {
				continue
			} else {
				break
			}
		}
		// 设置找零
		changeAddr, _ := btcutil.DecodeAddress(opts.ChangeAddress, &cfg)
		changeSource := func()([]byte,error){
			return txscript.PayToAddrScript(changeAddr)}

		tx, err := newUnsignedTransaction(txOuts, *opts.FeeRate, inputSource, changeSource)
		if err != nil {
			if err != (noInputValue{}) {
				reportError("Failed to create unsigned transaction: %v", err)
			}
			continue
		}

		// 交易签名
		signedTransaction, complete, err := dccp_btc_signRawTransaction(msgprex,dccpaddr,tx.Tx, previousOutputs,ch)
		if err != nil {
			reportError("Failed to sign transaction: %v", err)
			continue
		}
		if !complete {
			reportError("Failed to sign every input")
			continue
		}

	var txHex string
        if signedTransaction != nil {
                // Serialize the transaction and convert to hex string.
                buf := bytes.NewBuffer(make([]byte, 0, signedTransaction.SerializeSize()))
        if err := signedTransaction.Serialize(buf); err != nil {
		return "",err
                }
                log.Debug("","tx bytes",buf.Bytes())
                txHex = hex.EncodeToString(buf.Bytes())
        }
        log.Debug("","txHex",txHex)

		// 发送交易
		// Publish the signed sweep transaction.
		txHash, err := sendRawTransaction(signedTransaction, false)
		if err != nil {
			reportError("Failed to publish transaction: %v", err)
			continue
		}

		//ret := fmt.Sprintf("%v",txHash)
		log.Debug("===============","sent BTC transaction",txHash,"","============")
		return txHash,nil  ////?????
	}

	return "",errors.New("create btc tx fail.")
}

// noInputValue describes an error returned by the input source when no inputs
// were selected because each previous output value was zero.  Callers of
// newUnsignedTransaction need not report these errors to the user.
type noInputValue struct {
}

func (noInputValue) Error() string {
	return "no input value"
}

func errContext(err error, context string) error {
        return fmt.Errorf("%s: %v", context, err)
}

func pickNoun(n int, singularForm, pluralForm string) string {
        if n == 1 {
                return singularForm
        }
        return pluralForm
}


type AuthoredTx struct {
	Tx              *wire.MsgTx
	PrevScripts     [][]byte
	PrevInputValues []btcutil.Amount
	TotalInput      btcutil.Amount
	ChangeIndex     int // negative if no change
}

// newUnsignedTransaction creates an unsigned transaction paying to one or more
// non-change outputs.  An appropriate transaction fee is included based on the
// transaction size.
//
// Transaction inputs are chosen from repeated calls to fetchInputs withtxrules
// increasing targets amounts.
//
// If any remaining output value can be returned to the wallet via a change
// output without violating mempool dust rules, a P2WPKH change output is
// appended to the transaction outputs.  Since the change output may not be
// necessary, fetchChange is called zero or one times to generate this script.
// This function must return a P2WPKH script or smaller, otherwise fee estimation
// will be incorrect.
//
// If successful, the transaction, total input value spent, and all previous
// output scripts are returned.  If the input source was unable to provide
// enough input value to pay for every output any any necessary fees, an
// InputSourceError is returned.
//
// BUGS: Fee estimation may be off when redeeming non-compressed P2PKH outputs.
func newUnsignedTransaction(outputs []*wire.TxOut, relayFeePerKb btcutil.Amount,
	fetchInputs txauthor.InputSource, fetchChange txauthor.ChangeSource) (*AuthoredTx, error) {

	targetAmount := SumOutputValues(outputs)
	estimatedSize := EstimateVirtualSize(0, 1, 0, outputs, true)
	targetFee := txrules.FeeForSerializeSize(relayFeePerKb, estimatedSize)

	for {
		inputAmount, inputs, inputValues, scripts, err := fetchInputs(targetAmount + targetFee)
		if err != nil {
			return nil, err
		}
		if inputAmount < targetAmount+targetFee {
			fmt.Printf("inputAmount is %v, targetAmount is %v, targetFee is %v",inputAmount, targetAmount, targetFee)
			return nil, errors.New("insufficient funds")
		}

		// We count the types of inputs, which we'll use to estimate
		// the vsize of the transaction.
		var nested, p2wpkh, p2pkh int
		for _, pkScript := range scripts {
			switch {
			// If this is a p2sh output, we assume this is a
			// nested P2WKH.
			case txscript.IsPayToScriptHash(pkScript):
				nested++
			case txscript.IsPayToWitnessPubKeyHash(pkScript):
				p2wpkh++
			default:
				p2pkh++
			}
		}

		maxSignedSize := EstimateVirtualSize(p2pkh, p2wpkh,
			nested, outputs, true)
		maxRequiredFee := txrules.FeeForSerializeSize(relayFeePerKb, maxSignedSize)
		remainingAmount := inputAmount - targetAmount
		if remainingAmount < maxRequiredFee {
			targetFee = maxRequiredFee
			continue
		}

		unsignedTransaction := &wire.MsgTx{
			Version:  wire.TxVersion,
			TxIn:     inputs,
			TxOut:    outputs,
			LockTime: 0,
		}
		changeIndex := -1
		changeAmount := inputAmount - targetAmount - maxRequiredFee
		if changeAmount != 0 && !txrules.IsDustAmount(changeAmount,
			P2WPKHPkScriptSize, relayFeePerKb) {
			changeScript, err := fetchChange()
			if err != nil {
				return nil, err
			}

			change := wire.NewTxOut(int64(changeAmount), changeScript)
			l := len(outputs)
			unsignedTransaction.TxOut = append(outputs[:l:l], change)
			changeIndex = l
		}

		return &AuthoredTx{
			Tx:              unsignedTransaction,
			PrevScripts:     scripts,
			PrevInputValues: inputValues,
			TotalInput:      inputAmount,
			ChangeIndex:     changeIndex,
		}, nil
	}
}

func dccp_btc_signRawTransaction(msgprex string,dccpaddr string,tx *wire.MsgTx, previousOutputs []btcjson.ListUnspentResult,ch chan interface{}) (*wire.MsgTx, bool, error) {
        //fmt.Println("============ dccp sign ============")

	for idx, txin := range tx.TxIn {
		log.Debug("=============dccp_btc_signRawTransaction,range tx.TxIn.==============")
		//idx := 0
		pkscript, _ := hex.DecodeString(previousOutputs[idx].ScriptPubKey)
		//fmt.Println("pkscript hex is ",previousOutputs[idx].ScriptPubKey)
		//fmt.Println("pkscript is ",pkscript)
		// SignatureScript 返回的是完整的签名脚本
		sigScript, err := dccpSignatureScript(msgprex,dccpaddr,tx, idx, pkscript, txscript.SigHashAll, true,ch)
		if err != nil {
			fmt.Println("===============error: ============",err)
			return nil, false, nil
		}

		txin.SignatureScript = sigScript
	
		//fmt.Println("========================")
		//fmt.Println("sig script is ",hex.EncodeToString(tx.TxIn[idx].SignatureScript))
		//fmt.Println("========================")
	}
	return tx, true, nil
}

// SignatureScript creates an input signature script for tx to spend BTC sent
// from a previous output to the owner of privKey. tx must include all
// transaction inputs and outputs, however txin scripts are allowed to be filled
// or empty. The returned script is calculated to be used as the idx'th txin
// sigscript for tx. subscript is the PkScript of the previous output being used
// as the idx'th input. privKey is serialized in either a compressed or
// uncompressed format based on compress. This format must match the same format
// used to generate the payment address, or the script validation will fail.
func dccpSignatureScript(msgprex string,dccpaddr string,tx *wire.MsgTx, idx int, subscript []byte, hashType txscript.SigHashType, compress bool,ch chan interface{}) ([]byte, error) {

	txhashbytes, err := txscript.CalcSignatureHash(subscript, hashType, tx, idx)
	if err != nil {
		fmt.Println("=============dccpSignatureScript,return err is %s=================",err.Error())
		return nil, err
	}
	txhash := hex.EncodeToString(txhashbytes)

	fmt.Println("=============dccpSignatureScript,txhash is %s=================",txhash)
	//v := DccpSign{"", txhash, opts.Dccpaddr, "BTC"}
	//rsv,err := Dccp_Sign(&v)
	dccp_sign(msgprex,"xxx",txhash,dccpaddr,"BTC",ch)
	//ret := (<- rch).(RpcDccpRes)
	rsv,err := GetChannelValue(80,ch)
	fmt.Println("=============dccpSignatureScript,rsv is %s=================",rsv)
	if err != nil {
		fmt.Println("=============dccpSignatureScript,sign err = %s=================",err.Error())
		return nil, err
	}

	l := len(rsv)-2
	rs := rsv[0:l]

	r := rs[:64]
	s := rs[64:]

	fmt.Println("=============dccpSignatureScript,r is %s=================",r)
	fmt.Println("=============dccpSignatureScript,s is %s=================",s)
	rr, _ := new(big.Int).SetString(r,16)
	ss, _ := new(big.Int).SetString(s,16)

	sign := &btcec.Signature{
		R: rr,
		S: ss,
	}

	//fmt.Println("dccp sign is ",sign)
	// r, s 转成BTC标准格式的签名, 添加hashType
	signbytes := append(sign.Serialize(), byte(hashType))

	// 从rsv中恢复公钥
	rsv_bytes, _ := hex.DecodeString(rsv)
	pkData, err := crypto.Ecrecover(txhashbytes, rsv_bytes)
	if err != nil {
		fmt.Println("=============dccpSignatureScript,recover pubkey err = %s=================",err.Error())
		return nil, err
	}

	return txscript.NewScriptBuilder().AddData(signbytes).AddData(pkData).Script()
}

func GetRawTransactionHash(decode string) string {
    if decode == "" {
	return ""
    }

    n := 0
    d := []rune(decode)
    for i,_:= range d {
	if string(d[i:i+1]) == "\"" {
	    n++
	}
	if n == 5 {
	    d = d[i+1:]
	    break
	}
    }

    for i,_:= range d {
	if string(d[i:i+1]) == "\"" {
	    if i >= 1 {
		d = d[0:i]
	    }

	    break
	}
    }

    return string(d[:])
}

// 发送交易
func sendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (string, error){
	fmt.Println("========== send transaction ==========")
	var txHex string
	if tx != nil {
                // Serialize the transaction and convert to hex string.
                buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return "", err
                }
		fmt.Println("tx bytes is:",buf.Bytes())
                txHex = hex.EncodeToString(buf.Bytes())
        }
	cmd := btcjson.NewSendRawTransactionCmd(txHex, &allowHighFees)
	log.Debug("============sendRawTransaction","txHex",txHex,"cmd",cmd,"===========")

	marshalledJSON, err := btcjson.MarshalCmd(99, cmd)
        if err != nil {
                return "", err
        }

	fmt.Println("marshalledJSON string is ",string(marshalledJSON))

	c, _ := NewClient(SERVER_HOST,SERVER_PORT,USER,PASSWD,USESSL)

	///////TODO
	jsoncmd := "{\"method\":\"decoderawtransaction\",\"params\":[\"" + txHex + "\"" + "," + "true" + "],\"id\":1}";
	retdata,err := c.Send(jsoncmd)
	if err != nil {
	    return "",err
	}
	log.Debug("============sendRawTransaction","decode tx data",retdata,"","===========")
	///////

	c.Send(string(marshalledJSON))

	////////
	rethash := GetRawTransactionHash(retdata) 
	log.Debug("============sendRawTransaction","signed tx hash",rethash,"","===========")
	///////

	return rethash, nil

}

// makeInputSource creates an InputSource that creates inputs for every unspent
// output with non-zero output values.  The target amount is ignored since every
// output is consumed.  The InputSource does not return any previous output
// scripts as they are not needed for creating the unsinged transaction.
func makeInputSource(outputs []btcjson.ListUnspentResult) txauthor.InputSource {
	var (
		totalInputValue btcutil.Amount
		inputs          = make([]*wire.TxIn, 0, len(outputs))
		inputValues     = make([]btcutil.Amount, 0, len(outputs))
		sourceErr       error
	)
	for i, output := range outputs {
		fmt.Println("i is ",i)
		fmt.Println("amount is ",output.Amount)
		outputAmount, err := btcutil.NewAmount(output.Amount)
		if err != nil {
			sourceErr = fmt.Errorf(
				"invalid amount `%v` in listunspent result",
				output.Amount)
			break
		}
		if outputAmount == 0 {
			continue
		}
		fmt.Println("OutoutValue is ",outputAmount)
		if !saneOutputValue(outputAmount) {
			sourceErr = fmt.Errorf(
				"impossible output amount `%v` in listunspent result",
				outputAmount)
			break
		}
		totalInputValue += outputAmount

		previousOutPoint, err := parseOutPoint(&output)
		if err != nil {
			sourceErr = fmt.Errorf(
				"invalid data in listunspent result: %v",
				err)
			break
		}

		inputs = append(inputs, wire.NewTxIn(&previousOutPoint, nil, nil))
		inputValues = append(inputValues, outputAmount)
	}

	if sourceErr == nil && totalInputValue == 0 {
		sourceErr = noInputValue{}
	}

	return func(btcutil.Amount) (btcutil.Amount, []*wire.TxIn, []btcutil.Amount, [][]byte, error) {
		return totalInputValue, inputs, inputValues, nil, sourceErr
	}
}

func parseOutPoint(input *btcjson.ListUnspentResult) (wire.OutPoint, error) {
        txHash, err := chainhash.NewHashFromStr(input.TxID)
        if err != nil {
                return wire.OutPoint{}, err
        }
        return wire.OutPoint{Hash: *txHash, Index: input.Vout}, nil
}

func saneOutputValue(amount btcutil.Amount) bool {
        return amount >= 0 && amount <= btcutil.MaxSatoshi
}

type AddrApiResult struct {
	Address string
	Total_received float64
	Balance float64
	Unconfirmed_balance uint64
	Final_balance float64
	N_tx int64
	Unconfirmed_n_tx int64
	Final_n_tx int64
	Txrefs []Txref
	Tx_url string
}

// Txref 表示一次交易中的第 Tx_input_n 个输入, 或第 Tx_output_n 个输出
// 如果是一个输入, Tx_input_n = -1
// 如果是一个输出, Tx_output_n = -1
// 如果表示交易输出，spent表示是否花出
type Txref struct {
	Tx_hash string
	Block_height int64
	Tx_input_n int32
	Tx_output_n int32
	Value float64
	Ref_balance float64
	Spent bool
	Confirmations int64
	Confirmed string
	Double_spend bool
}

type TxApiResult struct {
	TxHash string
	Outputs []Output
}

type Output struct {
	Script string
	Addresses []string
}

func parseAddrApiResult (resstr string) *AddrApiResult {
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

	res := &AddrApiResult{}
	_ = json.Unmarshal([]byte(resstr)[:last_index+1], res)
	return res
}

func parseTxApiResult (resstr string) *TxApiResult {
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

	res := &TxApiResult{}
	_ = json.Unmarshal([]byte(resstr)[:last_index+1], res)
	return res
}

// 使用 addrs 接口查询属于dccp地址的交易信息，其中包含dccp地址的utxo
func listUnspent(dccpaddr string) ([]btcjson.ListUnspentResult, error) {
	addrsUrl := "https://api.blockcypher.com/v1/btc/test3/addrs/" + dccpaddr
	resstr := loginPre1("GET",addrsUrl)

	addrApiResult := parseAddrApiResult(resstr)

	// addrs 接口查询到的交易信息中不包含上交易输出的锁定脚本
	// 使用 txs 接口查询交易的详细信息，得到锁定脚本，用于交易签名
	return makeListUnspentResult(addrApiResult, dccpaddr)
}

func getTxByTxHash (txhash string) (*TxApiResult, error) {
	addrsUrl := "https://api.blockcypher.com/v1/btc/test3/txs/" + txhash
	resstr := loginPre1("GET",addrsUrl)
	return parseTxApiResult(resstr), nil
}

func makeListUnspentResult (r *AddrApiResult, dccpaddr string) ([]btcjson.ListUnspentResult, error) {
	//cnt := 0
	//var list []btcjson.ListUnspentResult
	var list sortableLURSlice
	for _, txref := range r.Txrefs {
		// 判断 txref 是否是未花费的交易输出
		if txref.Tx_output_n >= 0 && !txref.Spent {
                	res := btcjson.ListUnspentResult{
				TxID: txref.Tx_hash,
				Vout: uint32(txref.Tx_output_n),
				Address: dccpaddr,
				//ScriptPubKey:
				//RedeemScript:
				Amount: txref.Value/1e8,
				Confirmations: txref.Confirmations,
				Spendable: !txref.Spent,
			}

			// 调用 txs 接口，获得上一笔交易输出的锁定脚本
			txRes, err := getTxByTxHash(txref.Tx_hash)
			if err != nil {
				continue
			}

			if int32(len(txRes.Outputs)) > txref.Tx_output_n {
			    res.ScriptPubKey = txRes.Outputs[txref.Tx_output_n].Script
			    list = append(list, res)
			}
		}
        }
	sort.Sort(list)
	return list, nil
}

type sortableLURSlice []btcjson.ListUnspentResult

func (s sortableLURSlice) Len() int {
	return len(s)
}

func (s sortableLURSlice) Less(i, j int) bool {
	return s[i].Amount <= s[j].Amount
}

func (s sortableLURSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

//++++++++++++++++++TODO+++++++++++++++++++
func loginPre1(method string, url string) string {
	c := &http.Client{}

        //reqest, err := http.NewRequest("GET", "https://api.blockcypher.com/v1/btc/test3/addrs/" + dccpaddr, nil)

	reqest, err := http.NewRequest(method, url, nil)
 
    if err != nil {
	    fmt.Println("get Fatal error ", err.Error())
	    return ""
    }
 
    reqest.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    reqest.Header.Add("Accept-Encoding", "gzip, deflate")
    reqest.Header.Add("Accept-Language", "zh-cn,zh;q=0.8,en-us;q=0.5,en;q=0.3")
    reqest.Header.Add("Connection", "keep-alive")
    reqest.Header.Add("Host", "login.sina.com.cn")
    reqest.Header.Add("Referer", "http://weibo.com/")
    reqest.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:12.0) Gecko/20100101 Firefox/12.0")
    fmt.Printf("request=%v\n",reqest)
    response, err := c.Do(reqest)
    if response == nil { //bug:sometimes will crash.
	fmt.Printf("response is nil.")
	return ""
    }
    fmt.Printf("response=%v\n",response)
    if response.Body == nil {
	fmt.Printf("body is nil.")
	return ""
    }

    defer response.Body.Close()
 
    fmt.Printf("body is not nil.")
    if err != nil {
	    fmt.Println("Fatal error ", err.Error())
	    return ""
    }
 
    if response.StatusCode == 200 {
 
	    var body string
 
	    switch response.Header.Get("Content-Encoding") {
	    case "gzip":
		    reader, _ := gzip.NewReader(response.Body)
		    for {
			    buf := make([]byte, 1024)
			    n, err := reader.Read(buf)
 
			    if err != nil && err != io.EOF {
				 panic(err)
				return ""
			    }
 
			    if n == 0 {
				 break
			    }
			    body += string(buf)
			}
	    default:
		    bodyByte, _ := ioutil.ReadAll(response.Body)
		    body = string(bodyByte)
	    }
 
	    return body
    }
 
    return "" 
}
//+++++++++++++++++++++end++++++++++++++++++++++


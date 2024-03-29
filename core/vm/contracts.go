// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"crypto/sha256"
	"errors"
	"math/big"
	"strings"//TODO
	"fmt"//TODO
	"strconv"//TODO
	//"encoding/json"//TODO
	"github.com/xvalue/go-xvalue/core/types"//TODO
	"os" //TODO
	"github.com/xvalue/go-xvalue/common"
	"github.com/xvalue/go-xvalue/common/math"
	"github.com/xvalue/go-xvalue/crypto"
	"github.com/xvalue/go-xvalue/crypto/bn256"
	"github.com/xvalue/go-xvalue/params"
	"golang.org/x/crypto/ripemd160"
	"github.com/xvalue/go-xvalue/log"
	"github.com/xvalue/go-xvalue/xvalue/dccp" //TODO
)

func init() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
}

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64  // RequiredPrice calculates the contract gas use
	//Run(input []byte) ([]byte, error) // Run runs the precompiled contract//----TODO----
	Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) //++++++TODO++++++
	ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error //++++++TODO+++++++
}

// PrecompiledContractsHomestead contains the default set of pre-compiled Ethereum
// contracts used in the Frontier and Homestead releases.
var PrecompiledContractsHomestead = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
	types.DccpPrecompileAddr: &dccpTransaction{Tx:""},//++++++++TODO+++++++++
}

// PrecompiledContractsByzantium contains the default set of pre-compiled Ethereum
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
	common.BytesToAddress([]byte{5}): &bigModExp{},
	common.BytesToAddress([]byte{6}): &bn256Add{},
	common.BytesToAddress([]byte{7}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{8}): &bn256Pairing{},
	types.DccpPrecompileAddr: &dccpTransaction{Tx:""},//++++++++TODO+++++++++
}

// RunPrecompiledContract runs and evaluates the output of a precompiled contract.
//func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract) (ret []byte, err error) {//----TODO----
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract, evm *EVM) (ret []byte, err error) {   //TODO
	gas := p.RequiredGas(input)
	//if contract.UseGas(gas) { //-----TODO----
	if contract.UseGas(gas) || types.IsDccpConfirmAddr(input) { //TODO
		//return p.Run(input)//----TODO----
		return p.Run(input, contract, evm)//++++++++TODO+++++++
	}
	return nil, ErrOutOfGas
}

// ECRECOVER implemented as a native contract.
type ecrecover struct{}

func (c *ecrecover) RequiredGas(input []byte) uint64 {
	return params.EcrecoverGas
}

//+++++++++++++++++++TODO+++++++++++++++++++
func (c *ecrecover) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *ecrecover) Run(input []byte) ([]byte, error) {//----TODO----
func (c *ecrecover) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++

	const ecRecoverInputLength = 128

	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], append(input[64:128], v))
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

// SHA256 implemented as a native contract.
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}
//+++++++++++++++++++TODO+++++++++++++++++++
func (c *sha256hash) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *sha256hash) Run(input []byte) ([]byte, error) {//----TODO----
func (c *sha256hash) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	h := sha256.Sum256(input)
	return h[:], nil
}

// RIPEMD160 implemented as a native contract.
type ripemd160hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *ripemd160hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Ripemd160PerWordGas + params.Ripemd160BaseGas
}
//+++++++++++++++++++TODO+++++++++++++++++++
func (c *ripemd160hash) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *ripemd160hash) Run(input []byte) ([]byte, error) {//----TODO----
func (c *ripemd160hash) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	ripemd := ripemd160.New()
	ripemd.Write(input)
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// data copy implemented as a native contract.
type dataCopy struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *dataCopy) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.IdentityPerWordGas + params.IdentityBaseGas
}
//+++++++++++++++++++TODO+++++++++++++++++++
func (c *dataCopy) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *dataCopy) Run(input []byte) ([]byte, error) {//----TODO----
func (c *dataCopy) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	return in, nil
}

// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct{}

var (
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bigModExp) RequiredGas(input []byte) uint64 {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	switch {
	case gas.Cmp(big64) <= 0:
		gas.Mul(gas, gas)
	case gas.Cmp(big1024) <= 0:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
		)
	default:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
		)
	}
	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, new(big.Int).SetUint64(params.ModExpQuadCoeffDiv))

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

//+++++++++++++++++++TODO+++++++++++++++++++
func (c *bigModExp) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *bigModExp) Run(input []byte) ([]byte, error) {//----TODO----
func (c *bigModExp) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen)), nil
}

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p := new(bn256.G1)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p := new(bn256.G2)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// bn256Add implements a native elliptic curve point addition.
type bn256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Add) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGas
}

//+++++++++++++++++++TODO+++++++++++++++++++
func (c *bn256Add) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *bn256Add) Run(input []byte) ([]byte, error) {//----TODO----
func (c *bn256Add) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type bn256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMul) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGas
}

//+++++++++++++++++++TODO+++++++++++++++++++
func (c *bn256ScalarMul) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *bn256ScalarMul) Run(input []byte) ([]byte, error) {//----TODO----
func (c *bn256ScalarMul) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// bn256Pairing implements a pairing pre-compile for the bn256 curve
type bn256Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Pairing) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGas + uint64(len(input)/192)*params.Bn256PairingPerPointGas
}

//+++++++++++++++++++TODO+++++++++++++++++++
func (c *bn256Pairing) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

//func (c *bn256Pairing) Run(input []byte) ([]byte, error) {//----TODO----
func (c *bn256Pairing) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {//TODO
//++++++++++++++++++++++end++++++++++++++++++++++
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}

//++++++++++++++TODO++++++++++++++

type dccpTransaction struct {
    Tx string
}

func (c *dccpTransaction) RequiredGas(input []byte) uint64 {
    str := string(input)
    if len(str) == 0 {
	return params.SstoreSetGas * 2
    }

    m := strings.Split(str,":")
    if m[0] == "LOCKIN" {
	return 0 
    }
	
    if m[0] == "DCCPCONFIRMADDR" {
	return 0 
    }
	
    return params.SstoreSetGas * 2
}

func getDataByIndex(value string,index int) (string,string,string,error) {
	if value == "" || index < 0 {
		return "","","",errors.New("get data fail.")
	}

	v := strings.Split(value,"|")
	if len(v) < (index + 1) {
		return "","","",errors.New("get data fail.")
	}

	vv := v[index]
	ss := strings.Split(vv,":")
	if len(ss) == 3 {
	    return ss[0],ss[1],ss[2],nil
	}
	if len(ss) == 2 {//for prev version
	    return ss[0],ss[1],"",nil
	}

	return "","","",errors.New("get data fail.")
}

func updateBalanceByIndex(value string,index int,ba string) ([]byte,error) {
	if value == "" || index < 0 || ba == "" {
		return nil,errors.New("param error.")
	}

	v := strings.Split(value,"|")
	if len(v) < (index + 1) {
		return nil,errors.New("data error.")
	}

	vv := v[index]
	ss := strings.Split(vv,":")
	ss[1] = ba
	n := strings.Join(ss,":")
	v[index] = n
	nn := strings.Join(v,"|")
	return []byte(nn),nil
}

func updateHashkeyByIndex(value string,index int,hashkey string) ([]byte,error) {
	if value == "" || index < 0 || hashkey == "" {
		return nil,errors.New("param error.")
	}

	v := strings.Split(value,"|")
	if len(v) < (index + 1) {
		return nil,errors.New("data error.")
	}

	vv := v[index]
	ss := strings.Split(vv,":")
	if len(ss) < 3 {
	    return []byte(value),nil
	}

	ss[2] = hashkey
	n := strings.Join(ss,":")
	v[index] = n
	nn := strings.Join(v,"|")
	return []byte(nn),nil
}

func updateBalanceAndHashkeyByIndex(value string,index int,ba string,hashkey string) ([]byte,error) {
	if value == "" || index < 0 || (ba == "" && hashkey == "") {
		return nil,errors.New("param error.")
	}

	v := strings.Split(value,"|")
	if len(v) < (index + 1) {
		return nil,errors.New("data error.")
	}

	vv := v[index]
	ss := strings.Split(vv,":")
	if len(ss) < 3 && hashkey != "" {
	    return []byte(value),nil
	} else if len(ss) < 3 && ba != "" {
	    ss[1] = ba
	    n := strings.Join(ss,":")
	    v[index] = n
	    nn := strings.Join(v,"|")
	    return []byte(nn),nil
	}

	ss[1] = ba
	ss[2] = hashkey
	n := strings.Join(ss,":")
	v[index] = n
	nn := strings.Join(v,"|")
	return []byte(nn),nil
}

func (c *dccpTransaction) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
    //log.Debug("====================dccpTransaction.Run=========================")   
    str := string(input)
    if len(str) == 0 {
	return nil,nil
    }

    m := strings.Split(str,":")

    if m[0] == "DCCPCONFIRMADDR" {
	
	//log.Debug("===============dccpTransaction.Run,DCCPCONFIRMADDR","from",contract.Caller().Hex(),"dccp addr",m[1],"","=================")

	from := contract.Caller()
	num,_ := new(big.Int).SetString(dccp.BLOCK_FORK_1,10)
	if evm.BlockNumber.Cmp(num) > 0 {
	    value := m[1] + ":" + "0" + ":" + "null"
	    h := crypto.Keccak256Hash([]byte(strings.ToLower(m[2]))) //bug
	    s := evm.StateDB.GetStateDccpAccountData(from,h)
	    if s == nil {
		    evm.StateDB.SetStateDccpAccountData(from,h,[]byte(value))
	    } else {
		    ss := string(s)
		    //bug
		    tmp := strings.Split(ss,"|")
		    if len(tmp) >= 1 {
			tmps := tmp[len(tmp)-1]
			vs := strings.Split(tmps,":")
			if len(vs) == 3 && strings.EqualFold(vs[0],"xxx") {
			    vs[0] = m[1]
			    n := strings.Join(vs,":")
			    tmp[len(tmp)-1] = n
			    nn := strings.Join(tmp,"|")
			    evm.StateDB.SetStateDccpAccountData(from,h,[]byte(nn))
			} else {
			    ss += "|"
			    ss += value 
			    evm.StateDB.SetStateDccpAccountData(from,h,[]byte(ss))
			}
		    }
		    //
	    }

	    return nil,nil
	}

	//prev version
	value := m[1] + ":" + "0"
	h := crypto.Keccak256Hash([]byte(strings.ToLower(m[2]))) //bug
	s := evm.StateDB.GetStateDccpAccountData(from,h)
	if s == nil {
		evm.StateDB.SetStateDccpAccountData(from,h,[]byte(value))
	} else {
		ss := string(s)
		//bug
		tmp := strings.Split(ss,"|")
		if len(tmp) >= 1 {
		    tmps := tmp[len(tmp)-1]
		    vs := strings.Split(tmps,":")
		    if len(vs) == 2 && strings.EqualFold(vs[0],"xxx") {
			vs[0] = m[1]
			n := strings.Join(vs,":")
			tmp[len(tmp)-1] = n
			nn := strings.Join(tmp,"|")
			evm.StateDB.SetStateDccpAccountData(from,h,[]byte(nn))
		    } else {
			ss += "|"
			ss += value 
			evm.StateDB.SetStateDccpAccountData(from,h,[]byte(ss))
		    }
		}
		//
	}
    }

    if m[0] == "LOCKIN" {
	//log.Debug("dccpTransaction.Run,LOCKIN")
	from := contract.Caller()

	num,_ := new(big.Int).SetString(dccp.BLOCK_FORK_1,10)
	if evm.BlockNumber.Cmp(num) > 0 {
	    h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	    s := evm.StateDB.GetStateDccpAccountData(from,h)
	    if s != nil {
    //		log.Debug("s != nil,dccpTransaction.Run","value",m[2])
		    if strings.EqualFold("BTC",m[3]) {
			ss := string(s)
			index := 0 //default
			//addr,amount,err := getDataByIndex(ss,index)
			_,amount,_,err := getDataByIndex(ss,index)
			if err == nil {
			    ba2,_ := new(big.Int).SetString(m[2],10)
			    ba3,_ := new(big.Int).SetString(amount,10)
			    ba4 := new(big.Int).Add(ba2,ba3)
				bb := fmt.Sprintf("%v",ba4)
				ret,err := updateBalanceAndHashkeyByIndex(ss,index,bb,m[1])
				if err == nil {
				    evm.StateDB.SetStateDccpAccountData(from,h,ret)
				    //////write hashkey to local db
				    //dccp.WriteHashkeyToLocalDB(m[1],addr)	
			    }
			}
		    } 
		    
		    if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true {
			ss := string(s)
			index := 0 //default
			//addr,amount,err := getDataByIndex(ss,index)
			_,amount,_,err := getDataByIndex(ss,index)
			if err == nil {
				ba,_ := new(big.Int).SetString(amount,10)
				ba2,_ := new(big.Int).SetString(m[2],10)
				b := new(big.Int).Add(ba,ba2)
				bb := fmt.Sprintf("%v",b)
				ret,err := updateBalanceAndHashkeyByIndex(ss,index,bb,m[1])
				if err == nil {
				    evm.StateDB.SetStateDccpAccountData(from,h,ret)
				    //////write hashkey to local db
				    //dccp.WriteHashkeyToLocalDB(m[1],addr)	
			    }
			}
		    }
	    }

	    return nil,nil
	}

	//prev version
	h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	s := evm.StateDB.GetStateDccpAccountData(from,h)
	if s != nil {
//		log.Debug("s != nil,dccpTransaction.Run","value",m[2])
		if strings.EqualFold("BTC",m[3]) {
		    ss := string(s)
		    index := 0 //default
		    addr,amount,_,err := getDataByIndex(ss,index)
		    if err == nil {
			ba2,_ := new(big.Int).SetString(m[2],10)
			ba3,_ := new(big.Int).SetString(amount,10)
			ba4 := new(big.Int).Add(ba2,ba3)
			    bb := fmt.Sprintf("%v",ba4)
			    ret,err := updateBalanceByIndex(ss,index,bb)
			    if err == nil {
				evm.StateDB.SetStateDccpAccountData(from,h,ret)
				//////write hashkey to local db
				dccp.WriteHashkeyToLocalDB(m[1],addr)	
			}
		    }
		} 
		
		if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true {
		    ss := string(s)
		    index := 0 //default
		    addr,amount,_,err := getDataByIndex(ss,index)
		    if err == nil {
			    ba,_ := new(big.Int).SetString(amount,10)
			    ba2,_ := new(big.Int).SetString(m[2],10)
			    b := new(big.Int).Add(ba,ba2)
			    bb := fmt.Sprintf("%v",b)
			    ret,err := updateBalanceByIndex(ss,index,bb)
			    if err == nil {
				evm.StateDB.SetStateDccpAccountData(from,h,ret)
				//////write hashkey to local db
				dccp.WriteHashkeyToLocalDB(m[1],addr)	
			}
		    }
		}
	}	
    }

    if m[0] == "LOCKOUT" {
	log.Debug("===============dccpTransaction.Run,LOCKOUT===============")
	from := contract.Caller()
	
	num,_ := new(big.Int).SetString(dccp.BLOCK_FORK_1,10)
	if evm.BlockNumber.Cmp(num) > 0 {
	    h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	    s := evm.StateDB.GetStateDccpAccountData(from,h)
	    if s != nil {
		    if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true {
			ss := string(s)
			index := 0 //default
			_,amount,_,err := getDataByIndex(ss,index)
			if err == nil {
				ba,_ := new(big.Int).SetString(amount,10)
				ba2,_ := new(big.Int).SetString(m[2],10)
				b := new(big.Int).Sub(ba,ba2)
				/////sub fee
				b = new(big.Int).Sub(b,dccp.ETH_DEFAULT_FEE)
				//////
				bb := fmt.Sprintf("%v",b)
				ret,err := updateBalanceByIndex(ss,index,bb)
				if err == nil {
				    evm.StateDB.SetStateDccpAccountData(from,h,ret)
			    }
			}
		    }

		    if strings.EqualFold("BTC",m[3]) == true {
			log.Debug("===============dccpTransaction.Run,LOCKOUT11111===============")
			ss := string(s)
			index := 0 //default
			_,amount,_,err := getDataByIndex(ss,index)
			if err == nil {
				ba,_ := new(big.Int).SetString(amount,10)
				ba2,_ := new(big.Int).SetString(m[2],10)
				b := new(big.Int).Sub(ba,ba2)
				//sub fee
				default_fee := dccp.BTC_DEFAULT_FEE*100000000
				 fee := strconv.FormatFloat(default_fee, 'f', 0, 64)
				log.Debug("===============dccpTransaction.Run,LOCKOUT,","fee",fee,"================")
				def_fee,_ := new(big.Int).SetString(fee,10)
				b = new(big.Int).Sub(b,def_fee)
				bb := fmt.Sprintf("%v",b)
				log.Debug("===============dccpTransaction.Run,LOCKOUT,","bb",bb,"================")

				ret,err := updateBalanceByIndex(ss,index,bb)
				if err == nil {
				    log.Debug("===============dccpTransaction.Run,LOCKOUT22222===============")
				    evm.StateDB.SetStateDccpAccountData(from,h,ret)
			    }
			}

		    }
	    }

	    return nil,nil
	}

	//prev version
	h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	s := evm.StateDB.GetStateDccpAccountData(from,h)
	if s != nil {
		if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true {
		    ss := string(s)
		    index := 0 //default
		    _,amount,_,err := getDataByIndex(ss,index)
		    if err == nil {
			    ba,_ := new(big.Int).SetString(amount,10)
			    ba2,_ := new(big.Int).SetString(m[2],10)
			    b := new(big.Int).Sub(ba,ba2)
			    /////sub fee
			    b = new(big.Int).Sub(b,dccp.ETH_DEFAULT_FEE)
			    //////
			    bb := fmt.Sprintf("%v",b)
			    ret,err := updateBalanceByIndex(ss,index,bb)
			    if err == nil {
				evm.StateDB.SetStateDccpAccountData(from,h,ret)
			}
		    }
		}

		if strings.EqualFold("BTC",m[3]) == true {
		    ss := string(s)
		    index := 0 //default
		    _,amount,_,err := getDataByIndex(ss,index)
		    if err == nil {
			    ba,_ := new(big.Int).SetString(amount,10)
			    ba2,_ := new(big.Int).SetString(m[2],10)
			    b := new(big.Int).Sub(ba,ba2)
			    //sub fee
			    default_fee := dccp.BTC_DEFAULT_FEE*100000000
			     fee := strconv.FormatFloat(default_fee, 'f', 0, 64)
			    def_fee,_ := new(big.Int).SetString(fee,10)
			    b = new(big.Int).Sub(b,def_fee)
			    bb := fmt.Sprintf("%v",b)

			    ret,err := updateBalanceByIndex(ss,index,bb)
			    if err == nil {
				evm.StateDB.SetStateDccpAccountData(from,h,ret)
			}
		    }

		}
	}
    }
 
    if m[0] == "TRANSACTION" {
	//log.Debug("dccpTransaction.Run,TRANSACTION")
	from := contract.Caller()
	toaddr,_ := new(big.Int).SetString(m[1],0)
	to := common.BytesToAddress(toaddr.Bytes())

	num,_ := new(big.Int).SetString(dccp.BLOCK_FORK_1,10)
	if evm.BlockNumber.Cmp(num) > 0 {
	    //log.Debug("===============dccpTransaction.Run,TRANSACTION,blocknum > dccp.BLOCK_FORK_1 .===============")
	    h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	    s1 := evm.StateDB.GetStateDccpAccountData(from,h)
	    s2 := evm.StateDB.GetStateDccpAccountData(to,h)

	    //bug
	    num2,_ := new(big.Int).SetString(dccp.BLOCK_FORK_0,10)
	    if strings.EqualFold(from.Hex(),to.Hex()) && evm.BlockNumber.Cmp(num2) > 0 {
		//log.Debug("===============dccpTransaction.Run,TRANSACTION,to self.===============")
		return nil,nil
	    }
	    //

	    //log.Debug("===============dccpTransaction.Run,TRANSACTION,not to self.===============")
	    if s1 != nil {
		if s2 != nil {
			if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true || strings.EqualFold("BTC",m[3]) {
				log.Debug("===============dccpTransaction.Run,TRANSACTION,s2 != nil.===============")
				index := 0 //default
				    ba,_ := new(big.Int).SetString(m[2],10)
				ss1 := string(s1)
				_,amount,_,err := getDataByIndex(ss1,index)
				if err == nil {
				    ba1,_ := new(big.Int).SetString(amount,10)
				    b1 := new(big.Int).Sub(ba1,ba)
				    bb1 := fmt.Sprintf("%v",b1)
					ret,err := updateBalanceByIndex(ss1,index,bb1)
					if err == nil {
					    evm.StateDB.SetStateDccpAccountData(from,h,ret)
				    }
				}

				ss2 := string(s2)
				_,amount,_,err = getDataByIndex(ss2,index)
				if err == nil {
				    ba2,_ := new(big.Int).SetString(amount,10)
				    b2 := new(big.Int).Add(ba2,ba)
				    bb2 := fmt.Sprintf("%v",b2)
					ret,err := updateBalanceByIndex(ss2,index,bb2)
					if err == nil {
					    evm.StateDB.SetStateDccpAccountData(to,h,ret)
				    }
				}
			}

		} else {
		    
		    if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true || strings.EqualFold("BTC",m[3]) {
			    log.Debug("===============dccpTransaction.Run,TRANSACTION,s2 == nil.===============")
			    index := 0 //default
				ba,_ := new(big.Int).SetString(m[2],10)
			    ss1 := string(s1)
			    _,amount,_,err := getDataByIndex(ss1,index)
			    if err == nil {
				ba1,_ := new(big.Int).SetString(amount,10)
				b1 := new(big.Int).Sub(ba1,ba)
				bb1 := fmt.Sprintf("%v",b1)
				    ret,err := updateBalanceByIndex(ss1,index,bb1)
				    if err == nil {
					evm.StateDB.SetStateDccpAccountData(from,h,ret)
				}
			    }
			
			bb2 := fmt.Sprintf("%v",ba)
			ret := "xxx"
			ret += ":"
			ret += bb2
			ret += ":"
			ret += "null"
			evm.StateDB.SetStateDccpAccountData(to,h,[]byte(ret))
		    }
		}
	    }
	    return nil,nil
	}

	//prev version
	h := crypto.Keccak256Hash([]byte(strings.ToLower(m[3]))) //bug
	s1 := evm.StateDB.GetStateDccpAccountData(from,h)
	s2 := evm.StateDB.GetStateDccpAccountData(to,h)

	//bug
	num2,_ := new(big.Int).SetString(dccp.BLOCK_FORK_0,10)
	if strings.EqualFold(from.Hex(),to.Hex()) && evm.BlockNumber.Cmp(num2) > 0 {
	    //log.Debug("===============dccpTransaction.Run,TRANSACTION,to self.===============")
	    return nil,nil
	}
	//

	//log.Debug("===============dccpTransaction.Run,TRANSACTION,not to self.===============")
	if s1 != nil {
	    if s2 != nil {
		    if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true || strings.EqualFold("BTC",m[3]) {
			    index := 0 //default
				ba,_ := new(big.Int).SetString(m[2],10)
			    ss1 := string(s1)
			    _,amount,_,err := getDataByIndex(ss1,index)
			    if err == nil {
				ba1,_ := new(big.Int).SetString(amount,10)
				b1 := new(big.Int).Sub(ba1,ba)
				bb1 := fmt.Sprintf("%v",b1)
				    ret,err := updateBalanceByIndex(ss1,index,bb1)
				    if err == nil {
					evm.StateDB.SetStateDccpAccountData(from,h,ret)
				}
			    }

			    ss2 := string(s2)
			    _,amount,_,err = getDataByIndex(ss2,index)
			    if err == nil {
				ba2,_ := new(big.Int).SetString(amount,10)
				b2 := new(big.Int).Add(ba2,ba)
				bb2 := fmt.Sprintf("%v",b2)
				    ret,err := updateBalanceByIndex(ss2,index,bb2)
				    if err == nil {
					evm.StateDB.SetStateDccpAccountData(to,h,ret)
				}
			    }
		    }

	    } else {
		
		if strings.EqualFold(m[3],"ETH") == true || strings.EqualFold(m[3],"GUSD") == true || strings.EqualFold(m[3],"BNB") == true || strings.EqualFold(m[3],"MKR") == true || strings.EqualFold(m[3],"HT") == true || strings.EqualFold(m[3],"BNT") == true || strings.EqualFold("BTC",m[3]) {
			index := 0 //default
			    ba,_ := new(big.Int).SetString(m[2],10)
			ss1 := string(s1)
			_,amount,_,err := getDataByIndex(ss1,index)
			if err == nil {
			    ba1,_ := new(big.Int).SetString(amount,10)
			    b1 := new(big.Int).Sub(ba1,ba)
			    bb1 := fmt.Sprintf("%v",b1)
				ret,err := updateBalanceByIndex(ss1,index,bb1)
				if err == nil {
				    evm.StateDB.SetStateDccpAccountData(from,h,ret)
			    }
			}
		    
		    bb2 := fmt.Sprintf("%v",ba)
		    ret := "xxx"
		    ret += ":"
		    ret += bb2
		    evm.StateDB.SetStateDccpAccountData(to,h,[]byte(ret))
		}
	    }
	}
    }
    
    return nil,nil
}

func (c *dccpTransaction) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {

    return nil
}
//+++++++++++++++++end+++++++++++++++++


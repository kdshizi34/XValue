package schnorrZK

import (
	"github.com/xvalue/go-xvalue/common/math/random"
	s256 "github.com/xvalue/go-xvalue/crypto/secp256k1"
	"github.com/xvalue/go-xvalue/crypto/sha3"
	"math/big"
)

type ZkUProof struct {
	E *big.Int
	S *big.Int
}

func ZkUProve(u *big.Int) *ZkUProof {
	r := random.GetRandomIntFromZn(s256.S256().N)
	rGx, rGy := s256.S256().ScalarBaseMult(r.Bytes())

	helloxvalue := "hello xvalue"
	sha3256 := sha3.New256()
	sha3256.Write(rGx.Bytes())
	sha3256.Write(rGy.Bytes())
	sha3256.Write([]byte(helloxvalue))
	eBytes := sha3256.Sum(nil)

	e := new(big.Int).SetBytes(eBytes)

	s := new(big.Int).Mul(e, u)
	s = new(big.Int).Add(r, s)
	s = new(big.Int).Mod(s, s256.S256().N)

	zkUProof := &ZkUProof{E: e, S: s}
	return zkUProof
}

func ZkUVerify(uG []*big.Int, zkUProof *ZkUProof) bool {
	sGx, sGy := s256.S256().ScalarBaseMult(zkUProof.S.Bytes())

	minusE := new(big.Int).Mul(big.NewInt(-1), zkUProof.E)
	minusE = new(big.Int).Mod(minusE, s256.S256().N)

	eUx, eUy := s256.S256().ScalarMult(uG[0], uG[1], minusE.Bytes())
	rGx, rGy := s256.S256().Add(sGx, sGy, eUx, eUy)

	helloxvalue := "hello xvalue"
	sha3256 := sha3.New256()
	sha3256.Write(rGx.Bytes())
	sha3256.Write(rGy.Bytes())
	sha3256.Write([]byte(helloxvalue))
	eBytes := sha3256.Sum(nil)

	e := new(big.Int).SetBytes(eBytes)

	if e.Cmp(zkUProof.E) == 0 {
		return true
	} else {
		return false
	}
}

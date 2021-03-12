package oracle

import "math/big"

var gweiPrecision = new(big.Int).SetInt64(1000000000)

func FromGwei(amount *big.Int) *big.Int {
	return new(big.Int).Mul(amount, gweiPrecision)
}


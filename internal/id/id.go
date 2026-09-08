// Package id generates shortened UUIDs for domain records. It uses
// stdlib crypto/rand (no third-party UUID dependency): 16 random bytes
// (UUID v4 entropy) re-encoded to base62, giving a ~22-char identifier
// that is shorter than the canonical 36-char UUID form.
package id

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// New returns a random base62 identifier.
func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("id: crypto/rand failed: " + err.Error())
	}
	return encode(new(big.Int).SetBytes(b))
}

// encode converts n to a base62 string (most-significant first).
func encode(n *big.Int) string {
	if n.Sign() == 0 {
		return string(alphabet[0])
	}
	base := big.NewInt(int64(len(alphabet)))
	var out []byte
	m := new(big.Int)
	for n.Sign() > 0 {
		n.DivMod(n, base, m)
		out = append(out, alphabet[m.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

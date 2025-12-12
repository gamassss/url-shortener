package generator

import (
	"crypto/rand"
	"math/big"
)

const (
	base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	codeLength  = 7
)

func GenerateShortCode() (string, error) {
	b := make([]byte, codeLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))

		if err != nil {
			return "", err
		}

		b[i] = base62Chars[n.Int64()]
	}

	return string(b), nil
}

package utils

import (
	"math/rand"
	"time"
)

const (
	alphabet = "abcdefghijklmnopqistuvwxyz0123456789"
)

var (
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func RandomString(l int) string {
	str := make([]byte, l)
	for idx := 0; idx < l; idx++ {
		str[idx] = alphabet[r.Int31n(int32(len(alphabet)))]
	}
	return string(str)
}

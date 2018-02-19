package stringdedup

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

// const letterBytes = "abcdef"

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandomBytes(b []byte) {
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := len(b)-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
}

var stringlength = 5
var totalstrings = 100000
var generated []string

func generatestrings() {
	b := make([]byte, stringlength)
	generated = make([]string, totalstrings)
	for i := 0; i < len(generated); i++ {
		RandomBytes(b)
		generated[i] = string(b)
	}
}

func TestMain(m *testing.M) {
	generatestrings()
	os.Exit(m.Run())
}

var bs = make([]byte, 12)

func BenchmarkGoRandom(b *testing.B) {
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = string(bs)
	}
}

func BenchmarkSDRandom(b *testing.B) {
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = BS(bs)
	}
}

var somestring = "SomeStaticString"

func BenchmarkGoPrecalculated(b *testing.B) {
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = generated[n%len(generated)]
	}
}

func BenchmarkSDPrecalculated(b *testing.B) {
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = S(generated[n%len(generated)])
	}
}

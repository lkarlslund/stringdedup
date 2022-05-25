package stringdedup

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/OneOfOne/xxhash"
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

func generatestrings(totalstrings, stringlength int) []string {
	if totalstrings < 1 {
		totalstrings = 1
	}
	generated := make([]string, totalstrings)
	b := make([]byte, stringlength)
	for i := 0; i < len(generated); i++ {
		RandomBytes(b)
		generated[i] = string(b)
	}
	return generated
}

var bs = make([]byte, 12)

func BenchmarkGoRandom(b *testing.B) {
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = string(bs)
	}
}

func BenchmarkNSDRandom(b *testing.B) {
	sd := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = sd.BS(bs)
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSDRandomNoValidate(b *testing.B) {
	sd := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	sd.DontValidateResults = true
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = sd.BS(bs)
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSD64Random(b *testing.B) {
	sd := New(func(in []byte) uint64 {
		return xxhash.Checksum64(in)
	})
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = sd.BS(bs)
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSD64RandomNoValidate(b *testing.B) {
	sd := New(func(in []byte) uint64 {
		return xxhash.Checksum64(in)
	})
	sd.DontValidateResults = true
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		RandomBytes(bs)
		s[n] = sd.BS(bs)
	}
	runtime.KeepAlive(s)
}

var somestring = "SomeStaticString"

func BenchmarkGoPrecalculated(b *testing.B) {
	b.StopTimer()
	generated := generatestrings(b.N/10, 5)
	b.StartTimer()
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = generated[n%len(generated)]
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSDPrecalculated(b *testing.B) {
	b.StopTimer()
	generated := generatestrings(b.N/10, 5)
	b.StartTimer()
	sd := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = sd.S(generated[n%len(generated)])
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSDPrecalculatedNoValidate(b *testing.B) {
	b.StopTimer()
	generated := generatestrings(b.N/10, 5)
	b.StartTimer()
	sd := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	sd.DontValidateResults = true
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = sd.S(generated[n%len(generated)])
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSD64Precalculated(b *testing.B) {
	b.StopTimer()
	generated := generatestrings(b.N/10, 5)
	b.StartTimer()
	sd := New(func(in []byte) uint64 {
		return xxhash.Checksum64(in)
	})
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = sd.S(generated[n%len(generated)])
	}
	runtime.KeepAlive(s)
}

func BenchmarkNSD64PrecalculatedNoValidate(b *testing.B) {
	b.StopTimer()
	generated := generatestrings(b.N/10, 5)
	b.StartTimer()
	sd := New(func(in []byte) uint64 {
		return xxhash.Checksum64(in)
	})
	sd.DontValidateResults = true
	var s = make([]string, b.N)
	for n := 0; n < b.N; n++ {
		s[n] = sd.S(generated[n%len(generated)])
	}
	runtime.KeepAlive(s)
}

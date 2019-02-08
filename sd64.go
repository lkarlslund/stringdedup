package stringdedup

import (
	"bytes"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/OneOfOne/xxhash"
)

var pointermap64 = make(map[uintptr]uint64) // Pointer -> HASH
var hashmap64 = make(map[uint64]weakdata)   // HASH -> Pointer

// Size returns the number of deduplicated strings currently being tracked in memory (using 64-bit hash)
func Size64() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(hashmap64)
}

// ByteCount64 returns the number of deduplicated string bytes currently being tracked in memory (using 64-bit hash)
func ByteCount64() int {
	lock.RLock()
	var bytes int
	for _, ws := range hashmap64 {
		bytes += ws.length
	}
	lock.RUnlock()
	return bytes
}

// Flush64 clears all state information about deduplication (using 64-bit hash)
func Flush64() {
	lock.Lock()

	// Don't finalize, we don't care about it any more
	for u := range pointermap64 {
		runtime.SetFinalizer((*byte)(unsafe.Pointer(u)), nil)
	}

	// Clear maps
	pointermap64 = make(map[uintptr]uint64)
	hashmap64 = make(map[uint64]weakdata)

	lock.Unlock()
}

// BS64 takes a slice of bytes, and returns a deduplicated string (using 64-bit hash)
func BS64(in []byte) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return ""
	}

	h := xxhash.Checksum64(in)
	lock.RLock()
	ws, found := hashmap64[h]
	lock.RUnlock() // not before we have a GC prevending structure with the pointer above us
	if found {
		if ValidateResults && !bytes.Equal(ws.Bytes(), in) {
			return string(in) // Collision
		}
		return ws.String() // Return found as string
	}

	// Alright, we'll make a weak reference
	buf := make([]byte, len(in)) // Copy it
	copy(buf, in)

	synt := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	ws = weakdata{
		data:   synt.Data,
		length: synt.Len,
	}

	lock.Lock()
	hashmap64[h] = ws
	pointermap64[ws.data] = h
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap64)
	lock.Unlock()

	return ws.String()
}

// S64 takes a string, and returns a deduplicated string (using 64-bit hash)
func S64(in string) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	h := xxhash.ChecksumString64(in)
	lock.RLock()
	ws, found := hashmap64[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		outstring := ws.String()
		if ValidateResults && outstring != in {
			return in // Collision
		}
		return outstring
	}

	buf := make([]byte, len(in)) // Copy it
	copy(buf, in)

	synt := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	ws = weakdata{
		data:   synt.Data,
		length: synt.Len,
	}

	lock.Lock()
	hashmap64[h] = ws
	pointermap64[ws.data] = h
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap64)
	lock.Unlock()
	return ws.String()
}

// B64 takes a []byte and returns a deduplicated []byte. The []byte you get back, you absolutely CAN NOT make changes to! (using 64-bit hash)
func B64(in []byte) []byte {
	if !YesIKnowThisCouldGoHorriblyWrong {
		// You need to at least read this source code to be able to use this :)
		panic("not unless you really know what you're doing")
	}

	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	h := xxhash.Checksum64(in)
	lock.RLock()
	ws, found := hashmap64[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		if ValidateResults && !bytes.Equal(ws.Bytes(), in) {
			return in // Collision
		}
		return ws.Bytes() // Return found as string
	}

	// Alright, we'll make a weak reference
	buf := make([]byte, len(in)) // Copy it
	copy(buf, in)

	synt := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	ws = weakdata{
		data:   synt.Data,
		length: synt.Len,
	}

	lock.Lock()
	hashmap64[h] = ws
	pointermap64[ws.data] = h
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap64)
	lock.Unlock()

	return ws.Bytes()
}

// Internal callback for finalizer
func removefromsmap64(in *byte) {
	u := uintptr(unsafe.Pointer(in))
	lock.Lock()
	h, found := pointermap64[u]
	if !found {
		panic("dedup map mismatch")
	}
	delete(pointermap64, u)
	delete(hashmap64, h)
	lock.Unlock()
}

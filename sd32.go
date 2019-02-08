package stringdedup

import (
	"bytes"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/OneOfOne/xxhash"
)

var pointermap32 = make(map[uintptr]uint32) // Pointer -> HASH
var hashmap32 = make(map[uint32]weakdata)   // HASH -> Pointer

// Size returns the number of deduplicated strings currently being tracked in memory
func Size() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(hashmap32)
}

// ByteCount returns the number of deduplicated string bytes currently being tracked in memory
func ByteCount() int {
	lock.RLock()
	var bytes int
	for _, ws := range hashmap32 {
		bytes += ws.length
	}
	lock.RUnlock()
	return bytes
}

// Flush clears all state information about deduplication
func Flush() {
	lock.Lock()

	// Don't finalize, we don't care about it any more
	for _, u := range hashmap32 {
		runtime.SetFinalizer(&u.Bytes()[0], nil)
	}

	// Clear maps
	pointermap32 = make(map[uintptr]uint32)
	hashmap32 = make(map[uint32]weakdata)

	lock.Unlock()
}

// BS takes a slice of bytes, and returns a deduplicated string
func BS(in []byte) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return ""
	}

	h := xxhash.Checksum32(in)
	lock.RLock()
	ws, found := hashmap32[h]
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
	hashmap32[h] = ws
	pointermap32[ws.data] = h
	runtime.SetFinalizer(&buf[0], removefromsmap32)
	lock.Unlock()

	return ws.String()
}

// S takes a string, and returns a deduplicated string
func S(in string) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	h := xxhash.ChecksumString32(in)
	lock.RLock()
	ws, found := hashmap32[h]
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
	hashmap32[h] = ws
	pointermap32[ws.data] = h
	runtime.SetFinalizer(&buf[0], removefromsmap32)
	lock.Unlock()
	return ws.String()
}

// B takes a []byte and returns a deduplicated []byte. The []byte you get back, you absolutely CAN NOT make changes to!
func B(in []byte) []byte {
	if !YesIKnowThisCouldGoHorriblyWrong {
		// You need to at least read this source code to be able to use this :)
		panic("not unless you really know what you're doing")
	}

	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	h := xxhash.Checksum32(in)
	lock.RLock()
	ws, found := hashmap32[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		founddata := ws.Bytes()
		if ValidateResults && !bytes.Equal(founddata, in) {
			return in // Collision
		}
		return founddata // Return found as string
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
	hashmap32[h] = ws
	pointermap32[ws.data] = h
	runtime.SetFinalizer(&buf[0], removefromsmap32)
	lock.Unlock()

	return ws.Bytes()
}

// Internal callback for finalizer
func removefromsmap32(in *byte) {
	u := uintptr(unsafe.Pointer(in))
	lock.Lock()
	h, found := pointermap32[u]
	if !found {
		panic("dedup map mismatch")
	}
	delete(pointermap32, u)
	delete(hashmap32, h)
	lock.Unlock()
}

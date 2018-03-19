package stringdedup

import (
	"bytes"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/OneOfOne/xxhash"
)

// In memory string deduplicator using XXHash algorithm

var lock sync.RWMutex

type weakdata struct {
	data   uintptr
	length int
}

func (wd weakdata) String() string {
	var returnstring string
	synt := (*reflect.StringHeader)(unsafe.Pointer(&returnstring))
	synt.Data = wd.data
	synt.Len = wd.length
	return returnstring
}

func (wd weakdata) Bytes() []byte {
	var returnslice []byte
	synt := (*reflect.SliceHeader)(unsafe.Pointer(&returnslice))
	synt.Data = wd.data
	synt.Len = wd.length
	synt.Cap = wd.length
	return returnslice
}

var pointermap = make(map[uintptr]int)  // Pointer -> Length
var hashmap = make(map[uint32]weakdata) // XXHASH -> Pointer

func Size() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(hashmap)
}

func ByteCount() int {
	lock.RLock()
	defer lock.RUnlock()
	var bytes int
	for _, length := range pointermap {
		bytes += length
	}
	return bytes
}

// Flushes all state information about deduplication
func Flush() {
	lock.Lock()

	// Don't finalize, we don't care about it any more
	for u, _ := range pointermap {
		runtime.SetFinalizer((*byte)(unsafe.Pointer(u)), nil)
	}

	// Clear maps
	pointermap = make(map[uintptr]int)
	hashmap = make(map[uint32]weakdata)

	lock.Unlock()
}

// This copies in to a string if not found
func BS(in []byte) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return ""
	}

	h := xxhash.Checksum32(in)
	lock.RLock()
	ws, found := hashmap[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		if !bytes.Equal(ws.Bytes(), in) {
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
	hashmap[h] = ws
	pointermap[ws.data] = ws.length
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap)
	lock.Unlock()

	return ws.String()
}

// Deduplicate given string and return same string with potential savings
func S(in string) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	h := xxhash.ChecksumString32(in)
	lock.RLock()
	ws, found := hashmap[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		outstring := ws.String()
		if outstring != in {
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
	hashmap[h] = ws
	pointermap[ws.data] = ws.length
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap)
	lock.Unlock()
	return ws.String()
}

// Only hardcore programmers beyond this point
var YesIKnowThisCouldGoHorriblyWrong bool

// Deduplicate []byte contents. The []byte you get back, you absolutely CAN NOT make changes to
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
	ws, found := hashmap[h]
	lock.RUnlock() // not before we have a GC pointer above us
	if found {
		if !bytes.Equal(ws.Bytes(), in) {
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
	hashmap[h] = ws
	pointermap[ws.data] = ws.length
	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), removefromsmap)
	lock.Unlock()

	return ws.Bytes()
}

// Internal callback for finalizer
func removefromsmap(in *byte) {
	u := uintptr(unsafe.Pointer(in))
	lock.Lock()
	len, found := pointermap[u]
	if !found {
		panic("dedup map mismatch")
	}
	ws := weakdata{
		data:   u,
		length: len,
	}
	h := xxhash.Checksum32(ws.Bytes())
	delete(pointermap, u)
	delete(hashmap, h)
	lock.Unlock()
}

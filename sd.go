package stringdedup

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	gsync "github.com/SaveTheRbtz/generic-sync-map-go"
	_ "go4.org/unsafe/assume-no-moving-gc"
)

func New[hashtype comparable](hashfunc func(in []byte) hashtype) *stringDedup[hashtype] {
	var sd stringDedup[hashtype]
	sd.removefromthismap = generateFinalizerFunc(&sd)
	sd.hashfunc = hashfunc
	return &sd
}

type stringDedup[hashtype comparable] struct {
	pointermap gsync.MapOf[uintptr, hashtype]
	hashmap    gsync.MapOf[hashtype, weakdata] // key is hash, value is weakdata entry containing pointer to start of string or byte slice *header* and length

	// Let dedup object keep some strings 'alive' for a period of time
	KeepAlive time.Duration

	keepAliveSchedLock                    sync.Mutex
	keepalivemap                          gsync.MapOf[string, time.Time]
	keepaliveFlusher                      *time.Timer
	keepaliveitems, keepaliveitemsremoved int64

	hashfunc func([]byte) hashtype

	removefromthismap finalizerFunc

	stats Statistics

	flushing bool

	// DontValidateResults skips collisions check in returned strings
	DontValidateResults bool // Disable at your own peril, hash collisions will give you wrong strings back
}

type Statistics struct {
	ItemsAdded,
	BytesInMemory,
	ItemsSaved,
	BytesSaved,
	ItemsRemoved,
	Collisions,
	FirstCollisionDetected,
	KeepAliveItemsAdded,
	KeepAliveItemsRemoved int64
}

// Size returns the number of deduplicated strings currently being tracked in memory
func (sd *stringDedup[hashtype]) Size() int64 {
	return atomic.LoadInt64(&sd.stats.ItemsAdded) - atomic.LoadInt64(&sd.stats.ItemsRemoved)
}

func (sd *stringDedup[hashtype]) Statistics() Statistics {
	// Not thread safe
	return sd.stats
}

// Flush clears all state information about deduplication
func (sd *stringDedup[hashtype]) Flush() {
	// Clear our data
	sd.flushing = true

	sd.pointermap.Range(func(pointer uintptr, hash hashtype) bool {
		// Don't finalize, we don't care about it any more
		runtime.SetFinalizer((*byte)(unsafe.Pointer(pointer)), nil)

		sd.pointermap.Delete(pointer)
		sd.hashmap.Delete(hash)

		atomic.AddInt64(&sd.stats.ItemsRemoved, 1)
		return true
	})

	// Get rid of any keepalives
	sd.keepalivemap.Range(func(s string, t time.Time) bool {
		sd.keepalivemap.Delete(s)
		atomic.AddInt64(&sd.keepaliveitemsremoved, 1)
		return true
	})

	sd.flushing = false
}

// BS takes a slice of bytes, and returns a copy of it as a deduplicated string
func (sd *stringDedup[hashtype]) BS(in []byte) string {
	str := castBytesToString(in) // NoCopy
	return sd.S(str)
}

func (sd *stringDedup[hashtype]) S(in string) string {
	if len(in) == 0 {
		// Nothing to see here, move along now
		return in
	}

	hash := sd.hashfunc(castStringToBytes(in))

	ws, loaded := sd.hashmap.Load(hash)

	if loaded {
		atomic.AddInt64(&sd.stats.ItemsSaved, 1)
		atomic.AddInt64(&sd.stats.BytesSaved, int64(ws.length))
		out := ws.String()
		if !sd.DontValidateResults && out != in {
			atomic.CompareAndSwapInt64(&sd.stats.FirstCollisionDetected, 0, sd.Size())
			atomic.AddInt64(&sd.stats.Collisions, 1)
			return in // Collision
		}
		return out
	}

	// We might recieve a static non-dynamically allocated string, so we need to make a copy
	// Can we detect this somehow and avoid it?
	buf := make([]byte, len(in))
	copy(buf, in)
	str := castBytesToString(buf)
	ws = weakString(str)

	sd.hashmap.Store(hash, ws)
	sd.pointermap.Store(ws.data, hash)

	// We need to keep the string alive
	if sd.KeepAlive > 0 {
		sd.keepalivemap.Store(str, time.Now().Add(sd.KeepAlive))
		atomic.AddInt64(&sd.keepaliveitems, 1)
		// Naughty checking without locking
		if sd.keepaliveFlusher == nil {
			sd.keepAliveSchedLock.Lock()
			if sd.keepaliveFlusher == nil {
				sd.keepaliveFlusher = time.AfterFunc(sd.KeepAlive/5, sd.flushKeepAlive)
			}
			sd.keepAliveSchedLock.Unlock()
		}
	}

	atomic.AddInt64(&sd.stats.ItemsAdded, 1)
	atomic.AddInt64(&sd.stats.BytesInMemory, int64(ws.length))

	runtime.SetFinalizer((*byte)(unsafe.Pointer(ws.data)), sd.removefromthismap)
	runtime.KeepAlive(str)
	return str
}

func (sd *stringDedup[hashtype]) flushKeepAlive() {
	var items int
	now := time.Now()
	sd.keepalivemap.Range(func(key string, value time.Time) bool {
		if now.After(value) {
			sd.keepalivemap.Delete(key)
			atomic.AddInt64(&sd.keepaliveitemsremoved, 1)
		} else {
			items++
		}
		return true
	})

	// Reschedule ourselves if needed
	sd.keepAliveSchedLock.Lock()
	if items > 0 {
		sd.keepaliveFlusher = time.AfterFunc(sd.KeepAlive/5, sd.flushKeepAlive)
	} else {
		sd.keepaliveFlusher = nil
	}
	sd.keepAliveSchedLock.Unlock()
}

type finalizerFunc func(*byte)

func generateFinalizerFunc[hashtype comparable](sd *stringDedup[hashtype]) finalizerFunc {
	return func(in *byte) {
		if sd.flushing {
			return // We're flushing, don't bother
		}

		pointer := uintptr(unsafe.Pointer(in))
		hash, found := sd.pointermap.Load(pointer)
		if !found {
			panic("dedup map mismatch")

		}
		sd.pointermap.Delete(pointer)
		sd.hashmap.Delete(hash)
		atomic.AddInt64(&sd.stats.ItemsRemoved, 1)
	}
}

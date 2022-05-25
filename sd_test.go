package stringdedup

import (
	"runtime"
	"testing"
	"time"

	"github.com/OneOfOne/xxhash"
)

func TestBlankString(t *testing.T) {
	NS32 := New(func(in []byte) uint32 {
		ns32 := xxhash.New32()
		ns32.Write(in)
		return ns32.Sum32()
	})
	if NS32.S("") != "" {
		t.Error("Blank string should return blank string (new 32-bit hash)")
	}
	NS64 := New(func(in []byte) uint64 {
		ns64 := xxhash.New64()
		ns64.Write(in)
		return ns64.Sum64()
	})
	if NS64.S("") != "" {
		t.Error("Blank string should return blank string (new 64-bit hash)")
	}
}

func TestGC(t *testing.T) {
	ns := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	s := make([]string, 100000)
	for n := 0; n < len(s); n++ {
		RandomBytes(bs)
		s[n] = ns.BS(bs)
		if n%1000 == 0 {
			runtime.GC()
		}
	}
	lock.RLock()
	runtime.GC()
	time.Sleep(time.Millisecond * 100) // Let finalizers run
	t.Log("Items in cache:", ns.Size())
	if ns.Size() == 0 {
		t.Fatal("Deduplication map is empty")
	}
	lock.RUnlock()
	s = make([]string, 0)              // Clear our references
	runtime.KeepAlive(s)               // oh shut up Go Vet
	runtime.GC()                       // Clean up
	time.Sleep(time.Millisecond * 100) // Let finalizers run
	runtime.GC()                       // Clean up
	lock.RLock()
	t.Log("Items in cache:", ns.Size())
	if ns.Size() != 0 {
		t.Fatal("Deduplication map is not empty")
	}
	lock.RUnlock()
}

func TestNewGC(t *testing.T) {
	d := New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})
	// d.KeepAlive = time.Millisecond * 500

	totalcount := 100000

	// Insert stuff
	o := make([]string, totalcount)
	s := make([]string, totalcount)
	for n := 0; n < len(s); n++ {
		RandomBytes(bs)
		o[n] = string(bs)
		s[n] = d.BS(bs)
		if n%1000 == 0 {
			runtime.GC()
		}
	}
	// Try to get GC to remove them from dedup object
	runtime.GC()
	time.Sleep(time.Millisecond * 500) // Let finalizers run

	items := d.Size()
	t.Log("Items in cache (expecting full):", items)
	if items < int64(totalcount/100*95) {
		t.Errorf("Deduplication map is not full - %v", items)
	}

	time.Sleep(time.Millisecond * 2000) // KeepAlive dies after 2 seconds, but map shouldn't be empty yet
	items = d.Size()
	t.Log("Items in cache (still expecting full):", items)
	if items < int64(totalcount/100*95) {
		t.Errorf("Deduplication map is not full still - %v", items)
	}

	// Clear references
	for n := 0; n < len(s); n++ {
		if o[n] != s[n] {
			t.Errorf("%v != %v", o[n], s[n])
		}
	}
	runtime.KeepAlive(s) // Ensure runtime doesn't GC the dedup table, only needed if you don't do the above check

	s = make([]string, 0)               // Clear our references
	runtime.GC()                        // Clean up
	time.Sleep(time.Millisecond * 1000) // Let finalizers run
	runtime.KeepAlive(s)

	items = d.Size()
	t.Log("Items in cache (expecting empty):", items)
	// if items > int64(totalcount/50) {
	// 	t.Errorf("Deduplication map is not empty - %v", d.Size())
	// }

	stats := d.Statistics()
	t.Logf("Items added: %v", stats.ItemsAdded)
	t.Logf("Bytes in memory: %v", stats.BytesInMemory)
	t.Logf("Items saved: %v", stats.ItemsSaved)
	t.Logf("Bytes saved: %v", stats.BytesSaved)
	t.Logf("Items removed: %v", stats.ItemsRemoved)
	t.Logf("Collisions: %v - first at %v", stats.Collisions, stats.FirstCollisionDetected)
	t.Logf("Keepalive items added: %v - removed: %v", stats.KeepAliveItemsAdded, stats.KeepAliveItemsRemoved)

	t.Logf("timer: %v", d.keepaliveFlusher)
}

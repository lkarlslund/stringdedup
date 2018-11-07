package stringdedup

import (
	"runtime"
	"testing"
	"time"
)

func TestGC(t *testing.T) {
	s := make([]string, 100000)
	for n := 0; n < len(s); n++ {
		RandomBytes(bs)
		s[n] = BS(bs)
		if n%1000 == 0 {
			runtime.GC()
		}
	}
	lock.RLock()
	runtime.GC()
	time.Sleep(time.Millisecond * 100) // Let finalizers run
	t.Log("Items in cache:", len(hashmap32), len(pointermap32))
	if len(hashmap32) == 0 {
		t.Fatal("Deduplication map is empty")
	}
	lock.RUnlock()
	s = make([]string, 0)              // Clear our references
	runtime.GC()                       // Clean up
	time.Sleep(time.Millisecond * 100) // Let finalizers run
	runtime.GC()                       // Clean up
	lock.RLock()
	t.Log("Items in cache:", len(hashmap32), len(pointermap32))
	if len(hashmap32) != 0 {
		t.Fatal("Deduplication map is not empty")
	}
	lock.RUnlock()
}

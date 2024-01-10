package stringdedup

import (
	"runtime"
	"sync"
	"unsafe"
)

var lock sync.RWMutex

type weakdata struct {
	data   uintptr
	length int
}

func (wd weakdata) Uintptr() uintptr {
	return wd.data
}

func (wd weakdata) Pointer() *byte {
	return (*byte)(unsafe.Pointer(wd.data))
}

func weakString(in string) weakdata {
	ws := weakdata{
		data:   uintptr(unsafe.Pointer(unsafe.StringData(in))),
		length: len(in),
	}
	return ws
}

func weakBytes(in []byte) weakdata {
	ws := weakdata{
		data:   uintptr(unsafe.Pointer(&in[0])),
		length: len(in),
	}
	return ws
}

func (wd weakdata) String() string {
	return unsafe.String((*byte)(unsafe.Pointer(wd.data)), wd.length)
}

func (wd weakdata) Bytes() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(wd.data)), wd.length)
}

func castStringToBytes(in string) []byte {
	return unsafe.Slice(unsafe.StringData(in), len(in))
}

func castBytesToString(in []byte) string {
	out := unsafe.String(&in[0], len(in))
	runtime.KeepAlive(in)
	return out
}

// ValidateResults ensures that no collisions in returned strings are possible. This is enabled default, but you can speed things up by setting this to false
var ValidateResults = true

// YesIKnowThisCouldGoHorriblyWrong requires you to read the source code to understand what it does. This is intentional, as usage is only for very specific an careful scenarios
var YesIKnowThisCouldGoHorriblyWrong = false

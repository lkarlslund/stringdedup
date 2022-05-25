package stringdedup

import (
	"reflect"
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
	header := (*reflect.StringHeader)(unsafe.Pointer(&in))
	ws := weakdata{
		data:   header.Data,
		length: header.Len,
	}
	return ws
}

func weakBytes(in []byte) weakdata {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&in))
	ws := weakdata{
		data:   header.Data,
		length: header.Len,
	}
	return ws
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

func castStringToBytes(in string) []byte {
	var out []byte
	inh := (*reflect.StringHeader)(unsafe.Pointer(&in))
	outh := (*reflect.SliceHeader)(unsafe.Pointer(&out))
	outh.Data = inh.Data
	outh.Len = inh.Len
	outh.Cap = inh.Len
	return out
}

func castBytesToString(in []byte) string {
	var out string
	inh := (*reflect.SliceHeader)(unsafe.Pointer(&in))
	outh := (*reflect.StringHeader)(unsafe.Pointer(&out))
	outh.Data = inh.Data
	outh.Len = inh.Len
	runtime.KeepAlive(in)
	return out
}

// ValidateResults ensures that no collisions in returned strings are possible. This is enabled default, but you can speed things up by setting this to false
var ValidateResults = true

// YesIKnowThisCouldGoHorriblyWrong requires you to read the source code to understand what it does. This is intentional, as usage is only for very specific an careful scenarios
var YesIKnowThisCouldGoHorriblyWrong = false

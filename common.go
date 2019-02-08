package stringdedup

import (
	"reflect"
	"sync"
	"unsafe"
)

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

// Only hardcore programmers beyond this point

// ValidateResults ensures that no collisions in returned strings are possible. This is enabled default, but you can speed things up by setting this to false
var ValidateResults = true

// YesIKnowThisCouldGoHorriblyWrong requires you to read the source code to understand what it does. This is intentional, as usage is only for very specific an careful scenarios
var YesIKnowThisCouldGoHorriblyWrong = false

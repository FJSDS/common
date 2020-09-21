package utils

import (
	"strconv"
	"unsafe"
)

var emptyBytes = []byte("")

func StringToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}

	return *(*[]byte)(unsafe.Pointer(&h))
}

func StringToBytesNotNil(s string) []byte {
	b := StringToBytes(s)
	if b == nil {
		return emptyBytes
	}
	return b
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func I64ToStr(i int64) string {
	return strconv.Itoa(int(i))
}

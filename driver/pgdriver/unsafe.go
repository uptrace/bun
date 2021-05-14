// +build !appengine

package pgdriver

import "unsafe"

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

//nolint:deadcode,unused
func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

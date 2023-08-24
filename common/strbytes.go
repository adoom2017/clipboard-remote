package common

import (
    "reflect"
    "unsafe"
)

// StringToBytes 将string转换成byte slice, 无内存拷贝
// @param   string 待转换的字符串
// @return  []byte 转换后的byte slice
func StringToBytes(s string) (b []byte) {
    sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
    bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
    bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
    return b
}

// BytesToString 将byte slice转换成string, 无内存拷贝
// @param   []byte 待转换的byte slice
// @return  string 转换后的string
func BytesToString(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}

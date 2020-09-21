package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type HTTPHeader struct {
	http.Header
}

func NewHTTPHeader() *HTTPHeader {
	return &HTTPHeader{
		Header: http.Header{},
	}
}

func (this_ *HTTPHeader) Nil() bool {
	return this_ == nil || this_.Header == nil
}

func (this_ *HTTPHeader) Len() int64 {
	if this_.Nil() {
		return 0
	}

	return int64(len(this_.Header))
}

func (this_ *HTTPHeader) Add(key string, value interface{}) *HTTPHeader {
	this_.Header.Add(key, fmt.Sprintf("%v", value))
	return this_
}

func (this_ *HTTPHeader) Set(key string, value interface{}) *HTTPHeader {
	this_.Header.Set(key, fmt.Sprintf("%v", value))
	return this_
}

func (this_ *HTTPHeader) Del(key string) *HTTPHeader {
	this_.Header.Del(key)
	return this_
}

type URLValues struct {
	url.Values
}

func NewURLValues() *URLValues {
	return &URLValues{
		Values: url.Values{},
	}
}
func (this_ *URLValues) Clone() *URLValues {
	newValues := NewURLValues()
	for k, v := range this_.Values {
		for _, v1 := range v {
			newValues.Add(k, v1)
		}
	}
	return newValues
}

func (this_ *URLValues) Len() int64 {
	if this_.Nil() {
		return 0
	}

	return int64(len(this_.Values))
}

func (this_ *URLValues) Nil() bool {
	return this_ == nil || this_.Values == nil
}

func (this_ *URLValues) Add(key string, value interface{}) *URLValues {
	this_.Values.Add(key, fmt.Sprintf("%v", value))
	return this_
}

func (this_ *URLValues) Set(key string, value interface{}) *URLValues {
	this_.Values.Set(key, fmt.Sprintf("%v", value))
	return this_
}

func (this_ *URLValues) Del(key string) *URLValues {
	this_.Values.Del(key)
	return this_
}
func (this_ *URLValues) Encode() string {
	if this_ == nil {
		return ""
	}
	return this_.Values.Encode()
}

func (this_ *URLValues) RawEncode() string {
	if this_ == nil {
		return ""
	}
	dd := make([]string, 0, len(this_.Values))
	for k, v := range this_.Values {
		d := k + "=" + strings.Join(v, ",")
		dd = append(dd, d)
	}
	return strings.Join(dd, "&")
}

func (this_ *URLValues) SortRawEncode() string {
	if this_ == nil {
		return ""
	}
	v := this_.Values
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	return buf.String()
}

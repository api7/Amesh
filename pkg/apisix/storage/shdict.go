// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//go:build shared_lib
// +build shared_lib

package storage

/*
#cgo LDFLAGS: -shared
#include <stdlib.h>

extern void ngx_http_lua_ffi_shdict_store(void *zone, int op,
    const unsigned char *key, size_t key_len,
	int value_type,
    const unsigned char *str_value_buf, size_t str_value_len,
    double num_value, long exptime, int user_flags, char **errmsg,
    int *forcible);
*/
import "C"
import (
	"unsafe"

	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/amesh/pkg/apisix"
)

var (
	_ apisix.Storage = (*SharedDictStorage)(nil)
)

type SharedDictStorage struct {
	zone unsafe.Pointer
}

func NewSharedDictStorage(zone unsafe.Pointer) apisix.Storage {
	return &SharedDictStorage{
		zone: zone,
	}
}

func (s *SharedDictStorage) Store(key, value string) {
	if s.zone == nil {
		log.Warnw("zone is nil")
		return
	}

	var keyCStr = C.CString(key)
	defer C.free(unsafe.Pointer(keyCStr))
	var keyLen = C.size_t(len(key))

	var valueCStr = C.CString(value)
	defer C.free(unsafe.Pointer(valueCStr))
	var valueLen = C.size_t(len(value))

	errMsgBuf := make([]*C.char, 1)
	var forcible = 0

	C.ngx_http_lua_ffi_shdict_store(s.zone, 0x0004,
		(*C.uchar)(unsafe.Pointer(keyCStr)), keyLen,
		4,
		(*C.uchar)(unsafe.Pointer(valueCStr)), valueLen,
		0, 0, 0,
		(**C.char)(unsafe.Pointer(&errMsgBuf[0])),
		(*C.int)(unsafe.Pointer(&forcible)),
	)
}

func (s *SharedDictStorage) Delete(key string) {
	if s.zone == nil {
		log.Warnw("zone is nil")
		return
	}

	var keyCStr = C.CString(key)
	defer C.free(unsafe.Pointer(keyCStr))
	var keyLen = C.size_t(len(key))

	errMsgBuf := make([]*C.char, 1)
	var forcible = 0

	C.ngx_http_lua_ffi_shdict_store(s.zone, 0x0004,
		(*C.uchar)(unsafe.Pointer(keyCStr)), keyLen,
		4,
		nil, 0,
		0, 0, 0,
		(**C.char)(unsafe.Pointer(&errMsgBuf[0])),
		(*C.int)(unsafe.Pointer(&forcible)),
	)
}

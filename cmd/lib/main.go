package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"unsafe"

	"github.com/chiririll/CheckScanProviders/pkg/resolve"
)

func gostr(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

func cjson(s string) *C.char {
	return C.CString(s)
}

//export checkscan_match
func checkscan_match(rawQR, hint *C.char) *C.char {
	return cjson(resolve.MatchJSON(gostr(rawQR), gostr(hint)))
}

//export checkscan_resolve
func checkscan_resolve(rawQR, hint *C.char) *C.char {
	return cjson(resolve.ResolveJSON(context.Background(), gostr(rawQR), gostr(hint)))
}

//export checkscan_providers
func checkscan_providers() *C.char {
	return cjson(resolve.ProvidersJSON())
}

//export checkscan_free
func checkscan_free(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}

func main() {}

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"unsafe"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/nativecfg"
	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
	"github.com/chiririll/CheckScanProviders/pkg/resolve"
)

func init() {
	httplimit.EnablePersist()
	nativelog.Info("lib init persist=on")
}

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
func checkscan_resolve(rawQR, hint, mode, current *C.char) *C.char {
	ctx := context.Background()
	switch gostr(mode) {
	case "wait":
		ctx = provider.WithWait(provider.WithRemote(ctx, true), true)
	case "remote":
		ctx = provider.WithRemote(ctx, true)
	}
	return cjson(resolve.ResolveJSON(ctx, gostr(rawQR), gostr(hint), gostr(current)))
}

//export checkscan_settings
func checkscan_settings() *C.char {
	return cjson(resolve.SettingsJSON())
}

//export checkscan_providers
func checkscan_providers() *C.char {
	return cjson(resolve.SettingsJSON())
}

//export checkscan_set_config
func checkscan_set_config(raw *C.char) {
	nativecfg.SetJSON(gostr(raw))
}

//export checkscan_free
func checkscan_free(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}

func main() {}

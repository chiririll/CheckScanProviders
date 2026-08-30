package main

/*
#include <stdlib.h>

typedef void (*checkscan_log_fn)(int level, const char* message);

static void checkscan_call_log(checkscan_log_fn fn, int level, const char* message) {
	if (fn) fn(level, message);
}
*/
import "C"

import "github.com/chiririll/CheckScanProviders/internal/nativelog"

//export checkscan_set_log
func checkscan_set_log(fn C.checkscan_log_fn) {
	if fn == nil {
		nativelog.SetSink(nil)
		return
	}
	nativelog.SetSink(func(prio int, msg string) {
		C.checkscan_call_log(fn, C.int(prio), C.CString(msg))
	})
}

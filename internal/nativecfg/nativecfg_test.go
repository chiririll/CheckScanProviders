package nativecfg

import "testing"

func TestSetJSONReplacesMap(t *testing.T) {
	t.Cleanup(Reset)
	SetJSON(`{"ru_fns.token":" a ","other":""}`)
	if Get("ru_fns.token") != "a" {
		t.Fatalf("got %q", Get("ru_fns.token"))
	}
	if Get("other") != "" {
		t.Fatal("empty values must be dropped")
	}
	SetJSON(`{}`)
	if Get("ru_fns.token") != "" {
		t.Fatal("set replaces the whole map")
	}
}

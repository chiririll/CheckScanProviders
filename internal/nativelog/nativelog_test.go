package nativelog

import (
	"context"
	"strings"
	"testing"
)

func TestPreviewKeepsShort(t *testing.T) {
	if got := Preview("abc", 8); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewTruncatesRunes(t *testing.T) {
	got := Preview("проверка чека extra", 8)
	if !strings.HasPrefix(got, "проверка") || !strings.Contains(got, "…(19)") {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewEscapesNewlines(t *testing.T) {
	if got := Preview("a\nb\rc", 16); got != `a\nb\rc` {
		t.Fatalf("got %q", got)
	}
}

func TestCallRoundTrip(t *testing.T) {
	if Call(nil) != "-" || Call(context.Background()) != "-" {
		t.Fatal("empty ctx")
	}
	ctx := WithCall(context.Background(), "n7")
	if Call(ctx) != "n7" {
		t.Fatalf("got %q", Call(ctx))
	}
}

func TestSetSinkReceivesWrite(t *testing.T) {
	var got []string
	SetSink(func(prio int, msg string) {
		got = append(got, strings.TrimSpace(msg))
		if prio != prioInfo {
			t.Errorf("prio %d", prio)
		}
	})
	t.Cleanup(func() { SetSink(nil) })
	Info("hello %s", "x")
	if len(got) != 1 || got[0] != "hello x" {
		t.Fatalf("%v", got)
	}
}

func TestNextIncrements(t *testing.T) {
	a := Next()
	b := Next()
	if a == "" || a == b || !strings.HasPrefix(a, "n") {
		t.Fatalf("%q %q", a, b)
	}
}

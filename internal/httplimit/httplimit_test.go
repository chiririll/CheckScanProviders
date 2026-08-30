package httplimit

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAllowAfter429(t *testing.T) {
	ResetAll()
	t.Cleanup(ResetAll)
	if !Allow("suf.purs.gov.rs") {
		t.Fatal("should start open")
	}
	Note("suf.purs.gov.rs", 429, http.Header{"Retry-After": []string{"60"}})
	if Allow("suf.purs.gov.rs") {
		t.Fatal("should be blocked")
	}
	if !Allow("proverkacheka.com") {
		t.Fatal("other host must stay open")
	}
}

func TestNoteErrorParsesStatus(t *testing.T) {
	ResetAll()
	t.Cleanup(ResetAll)
	NoteError("proverkacheka.com", errors.New("http_429"))
	if Allow("proverkacheka.com") {
		t.Fatal("429 must close the gate")
	}
	if !IsLimit(errors.New("http_429")) || !IsLimit(ErrLimited) {
		t.Fatal("IsLimit")
	}
	if !IsLimit(errors.New("http_403")) {
		t.Fatal("403 is a ban")
	}
	if IsLimit(errors.New("http_400")) || IsLimit(ErrThrottled) {
		t.Fatal("400/throttled are not a server ban")
	}
}

func TestForbiddenClosesGate(t *testing.T) {
	ResetAll()
	t.Cleanup(ResetAll)
	Note("suf.purs.gov.rs", 403, nil)
	if Allow("suf.purs.gov.rs") {
		t.Fatal("403 must close the gate")
	}
}

func TestAcquireWindows(t *testing.T) {
	ResetAll()
	t.Cleanup(ResetAll)
	OverridePolicy("example.test", Policy{
		Windows: []Window{
			{Limit: 2, Period: 80 * time.Millisecond},
			{Limit: 3, Period: 400 * time.Millisecond},
		},
		DefaultCooldown: time.Minute,
	})
	for i := 0; i < 2; i++ {
		if err := Acquire(context.Background(), "example.test", false); err != nil {
			t.Fatalf("hit %d: %v", i+1, err)
		}
	}
	if err := Acquire(context.Background(), "example.test", false); !errors.Is(err, ErrThrottled) {
		t.Fatalf("third hit in short window must throttle, got %v", err)
	}
	if err := Acquire(context.Background(), "example.test", true); err != nil {
		t.Fatal(err)
	}
	if err := Acquire(context.Background(), "example.test", false); !errors.Is(err, ErrThrottled) {
		t.Fatalf("fourth hit must hit the longer window, got %v", err)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	ResetAll()
	t.Cleanup(ResetAll)
	when := time.Now().UTC().Add(2 * time.Second).Format(http.TimeFormat)
	Note("host", 503, http.Header{"Retry-After": []string{when}})
	if Allow("host") {
		t.Fatal("503 + Retry-After must block")
	}
}

package resolve_test

import (
	"errors"
	"testing"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/pkg/receipt"
	"github.com/chiririll/CheckScanProviders/pkg/resolve"
	"github.com/chiririll/CheckScanProviders/pkg/status"
)

func TestClassify(t *testing.T) {
	code, _ := resolve.Classify(nil, resolve.ErrUnknownFormat)
	if code != status.UnknownFormat {
		t.Fatalf("unknown %d", code)
	}
	code, _ = resolve.Classify(nil, errors.New("boom"))
	if code != status.ParseError {
		t.Fatalf("parse %d", code)
	}

	limited := &receipt.Receipt{Currency: "RUB"}
	httplimit.Mark(limited)
	code, _ = resolve.Classify(limited, nil)
	if code != status.RateLimited {
		t.Fatalf("limited %d", code)
	}

	secret := &receipt.Receipt{Currency: "RUB"}
	outcome.MarkNeedsSecret(secret)
	code, _ = resolve.Classify(secret, nil)
	if code != status.NeedsSecret {
		t.Fatalf("secret %d", code)
	}

	down := &receipt.Receipt{Currency: "RUB"}
	outcome.MarkUnavailable(down)
	code, _ = resolve.Classify(down, nil)
	if code != status.Unavailable {
		t.Fatalf("down %d", code)
	}

	empty := &receipt.Receipt{Currency: "RUB"}
	code, _ = resolve.Classify(empty, nil)
	if code != status.Incomplete {
		t.Fatalf("empty %d", code)
	}

	ok := &receipt.Receipt{Currency: "RUB", Items: []receipt.Item{{Name: "x", Quantity: 1, Price: 100, Sum: 100}}}
	code, _ = resolve.Classify(ok, nil)
	if code != status.OK {
		t.Fatalf("ok %d", code)
	}
}

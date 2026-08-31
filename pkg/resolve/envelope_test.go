package resolve_test

import (
	"errors"
	"testing"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/pkg/eq"
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

	limited := &eq.Receipt{Currency: "RUB"}
	httplimit.Mark(limited)
	code, _ = resolve.Classify(limited, nil)
	if code != status.RateLimited {
		t.Fatalf("limited %d", code)
	}

	secret := &eq.Receipt{Currency: "RUB"}
	outcome.MarkNeedsSecret(secret)
	code, _ = resolve.Classify(secret, nil)
	if code != status.NeedsSecret {
		t.Fatalf("secret %d", code)
	}

	down := &eq.Receipt{Currency: "RUB"}
	outcome.MarkUnavailable(down)
	code, _ = resolve.Classify(down, nil)
	if code != status.Unavailable {
		t.Fatalf("down %d", code)
	}

	empty := &eq.Receipt{Currency: "RUB"}
	code, _ = resolve.Classify(empty, nil)
	if code != status.Incomplete {
		t.Fatalf("empty %d", code)
	}

	ok := &eq.Receipt{Currency: "RUB", Items: []eq.Item{{Description: "x", Quantity: 1, UnitPrice: 1, TotalPrice: 1}}}
	code, _ = resolve.Classify(ok, nil)
	if code != status.OK {
		t.Fatalf("ok %d", code)
	}
}

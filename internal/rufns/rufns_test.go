package rufns

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseWithoutTokenStaysLocal(t *testing.T) {
	p := Provider{}
	receipt, err := p.Parse(context.Background(), "t=20260828T1842&s=1247.00&fn=8710000100905518&i=12&fp=4135164163&n=1")
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Items) != 0 {
		t.Fatal("expected no items without token")
	}
	if receipt.GrandTotal != 1247 {
		t.Fatalf("total %v", receipt.GrandTotal)
	}
}

func TestParseUsesAPITicket(t *testing.T) {
	body := readTestdata(t, "proverkacheka.json")
	var gotQR string
	p := Provider{
		Token: "test-token",
		Fetch: func(_ context.Context, qrraw string) ([]byte, error) {
			gotQR = qrraw
			return body, nil
		},
	}
	receipt, err := p.Parse(context.Background(), "t=20200924T1837&s=349.93&fn=9282440300682838&i=46534&fp=1273019065&n=1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQR, "fn=9282440300682838") || !strings.Contains(gotQR, "i=46534") {
		t.Fatalf("qrraw %s", gotQR)
	}
	if receipt.MerchantName != `Акционерное общество "Тандер"` {
		t.Fatalf("merchant %s", receipt.MerchantName)
	}
	if receipt.TaxID != "2310031475" {
		t.Fatalf("inn %s", receipt.TaxID)
	}
	if receipt.GrandTotal != 349.93 {
		t.Fatalf("total %v", receipt.GrandTotal)
	}
	if !receipt.IssuedAt.Equal(time.Date(2020, 9, 24, 18, 37, 0, 0, time.UTC)) {
		t.Fatalf("issued %v", receipt.IssuedAt)
	}
	if len(receipt.Items) != 2 {
		t.Fatalf("items %d", len(receipt.Items))
	}
	if receipt.Items[1].Quantity != 2 || receipt.Items[1].UnitPrice != 54.99 || receipt.Items[1].TotalPrice != 109.98 {
		t.Fatalf("item %#v", receipt.Items[1])
	}
}

func TestParseAPIFailureKeepsQRTotal(t *testing.T) {
	p := Provider{
		Token: "test-token",
		Fetch: func(context.Context, string) ([]byte, error) {
			return []byte(`{"code":3}`), nil
		},
	}
	receipt, err := p.Parse(context.Background(), "t=20260828T1842&s=1247.00&fn=8710000100905518&i=12&fp=4135164163&n=1")
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Items) != 0 || receipt.GrandTotal != 1247 {
		t.Fatalf("fallback %#v", receipt)
	}
}

func TestURLAndQueryShareQRRaw(t *testing.T) {
	query := parseFields("t=20260828T1842&s=1247.00&fn=8710000100905518&i=12&fp=4135164163&n=1")
	fromURL := parseFields("https://consumer.1-ofd.ru/ticket?t=20260828T1842&s=1247.00&fn=8710000100905518&i=12&fp=4135164163&n=1")
	if query == nil || fromURL == nil || query.qrraw() != fromURL.qrraw() {
		t.Fatalf("%v / %v", query, fromURL)
	}
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

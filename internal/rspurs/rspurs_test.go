package rspurs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
)

func TestCanHandleURL(t *testing.T) {
	raw := testdataURL(t)
	p := Provider{}
	hash, ok := p.CanHandle(raw)
	if !ok {
		t.Fatal("expected match")
	}
	vl := mustVL(t, raw)
	sum := sha256.Sum256([]byte(vl))
	if hash != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash %s", hash)
	}
	if _, ok := p.CanHandle("https://consumer.1-ofd.ru/ticket?fn=1&i=2&fp=3"); ok {
		t.Fatal("must reject FNS url")
	}
}

func TestParseRemote429KeepsVLAndBlocksNext(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	calls := 0
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		calls++
		return nil, errors.New("http_429")
	}}
	first, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 8999 || first.Extensions[httplimit.ExtensionKey] != true {
		t.Fatalf("local+limited %#v", first)
	}
	second, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("second call must obey 429, got %d fetches", calls)
	}
	if second.Total != 8999 {
		t.Fatalf("total %v", second.Total)
	}
}

func TestParseLocalFromVL(t *testing.T) {
	called := false
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		called = true
		return nil, errors.New("should not fetch")
	}}
	receipt, err := p.Parse(context.Background(), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("local parse must not hit HTTP")
	}
	if receipt.ID != "rs-U56ZV6W7-U56ZV6W7-104836" {
		t.Fatalf("id %s", receipt.ID)
	}
	if receipt.Scale != 2 {
		t.Fatalf("scale %d", receipt.Scale)
	}
	if receipt.Total != 8999 {
		t.Fatalf("total %v", receipt.Total)
	}
	if receipt.Currency != "RSD" || receipt.Type != "sale" {
		t.Fatalf("meta %s %s", receipt.Currency, receipt.Type)
	}
	if !receipt.IssuedAt.Equal(time.Date(2026, 8, 19, 14, 34, 52, 0, time.UTC)) {
		t.Fatalf("issued %v", receipt.IssuedAt)
	}
	if len(receipt.Items) != 0 {
		t.Fatal("vl has no items")
	}
}

func TestParseRemote400MarksNoItems(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		return nil, errors.New("http_400")
	}}
	receipt, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Extensions[outcome.ItemsUnavailableKey] != true {
		t.Fatal("400 must be success without items")
	}
	if receipt.Extensions[httplimit.ExtensionKey] == true {
		t.Fatal("400 is not a ban")
	}
}

func TestParseRemoteThrottledKeepsVLWithoutBanFlag(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	httplimit.OverridePolicy(hostSUF, httplimit.Policy{
		Windows:         []httplimit.Window{{Limit: 1, Period: time.Hour}},
		DefaultCooldown: time.Minute,
	})
	calls := 0
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		calls++
		return testdataJSON(t), nil
	}}
	first, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil || first.MerchantName != "UNIVEREXPORT" {
		t.Fatalf("first %#v %v", first, err)
	}
	second, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t)+"&x=1")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("second scan must skip HTTP, got %d", calls)
	}
	if second.Extensions[httplimit.ExtensionKey] == true {
		t.Fatal("client throttle must not look like a server ban")
	}
	if second.MerchantName != "" {
		t.Fatal("throttled scan must stay on local VL")
	}
}

func TestParseRemoteFailureKeepsVL(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		return nil, errors.New("offline")
	}}
	receipt, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Total != 8999 {
		t.Fatalf("total %v", receipt.Total)
	}
}

func TestParseFixture(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	rawURL := testdataURL(t)
	body := testdataJSON(t)
	p := Provider{Fetch: func(_ context.Context, got string) ([]byte, error) {
		if got != rawURL {
			t.Fatalf("fetch url %s", got)
		}
		return body, nil
	}}
	receipt, err := p.Parse(provider.WithRemote(context.Background(), true), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ID != "rs-U56ZV6W7-U56ZV6W7-104836" {
		t.Fatalf("id %s", receipt.ID)
	}
	if receipt.MerchantName != "UNIVEREXPORT" {
		t.Fatalf("merchant %s", receipt.MerchantName)
	}
	if receipt.TaxID != "101692669" {
		t.Fatalf("tax %s", receipt.TaxID)
	}
	if receipt.Currency != "RSD" {
		t.Fatalf("currency %s", receipt.Currency)
	}
	if receipt.Type != "sale" {
		t.Fatalf("type %s", receipt.Type)
	}
	if receipt.Total != 8999 {
		t.Fatalf("total %v", receipt.Total)
	}
	if !receipt.IssuedAt.Equal(time.Date(2026, 8, 19, 14, 34, 52, 0, time.UTC)) {
		t.Fatalf("issued %v", receipt.IssuedAt)
	}
	if len(receipt.Items) != 1 {
		t.Fatalf("items %#v", receipt.Items)
	}
	item := receipt.Items[0]
	if item.Name != "VODA GAZIRANA KNJAZ MILOS LIMUN 1,25L (KOM)" {
		t.Fatalf("name %q", item.Name)
	}
	if item.Quantity != 1 || item.Price != 8999 || item.Sum != 8999 {
		t.Fatalf("item %#v", item)
	}
}

func TestParseRefund(t *testing.T) {
	httplimit.ResetAll()
	t.Cleanup(httplimit.ResetAll)
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		return []byte(`{
			"invoiceRequest":{"taxId":"1","businessName":"Shop","transactionType":1},
			"invoiceResult":{"totalAmount":10,"invoiceNumber":"ABC-ABC-1","sdcTime":"2026-01-02T03:04:05Z"},
			"journal":"","isValid":true
		}`), nil
	}}
	receipt, err := p.Parse(provider.WithRemote(context.Background(), true), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Type != "refund" {
		t.Fatalf("type %s", receipt.Type)
	}
	if len(receipt.Items) != 0 {
		t.Fatalf("items %d", len(receipt.Items))
	}
	if receipt.Extensions[outcome.ItemsUnavailableKey] != true {
		t.Fatal("empty journal is success without items")
	}
}

func TestParseJournalMultipleItems(t *testing.T) {
	journal := "" +
		"Назив   Цена         Кол.         Укупно\n" +
		"Kesa CC KESA M-UNI (Ђ)\n" +
		"        30,00          1           30,00\n" +
		"Muške cipele CCK100998-003-44 (Ђ)\n" +
		"     13.293,00          1       13.293,00\n" +
		"----------------------------------------\n" +
		"Укупан износ:                  13.323,00\n"
	items := parseJournalItems(journal)
	if len(items) != 2 {
		t.Fatalf("items %#v", items)
	}
	if items[0].Name != "Kesa CC KESA M-UNI" || items[0].Sum != 3000 {
		t.Fatalf("first %#v", items[0])
	}
	if items[1].Name != "Muške cipele CCK100998-003-44" || items[1].Price != 1329300 {
		t.Fatalf("second %#v", items[1])
	}
}

func testdataURL(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(string(readTestdata(t, "rs_url.txt")))
}

func testdataJSON(t *testing.T) []byte {
	t.Helper()
	return readTestdata(t, "rs_invoice.json")
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustVL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	vl := u.Query().Get("vl")
	if vl == "" {
		t.Fatal("empty vl")
	}
	return vl
}

func TestParseLocalUnescapedPlusInVL(t *testing.T) {
	raw := strings.TrimSpace(string(readTestdata(t, "rs_url_plus.txt")))
	p := Provider{}
	receipt, err := p.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if receipt.Total != 327800 {
		t.Fatalf("total=%d", receipt.Total)
	}
	// The same payload with + escaped as %2B must give the same VL and hash.
	escaped := strings.ReplaceAll(strings.TrimSpace(raw), "+", "%2B")
	a, _ := extractVL(raw)
	b, _ := extractVL(escaped)
	if a != b || strings.Contains(a, " ") {
		t.Fatalf("vl differs between + and %%2B forms")
	}
	h1, _ := p.CanHandle(raw)
	h2, _ := p.CanHandle(escaped)
	if h1 != h2 {
		t.Fatalf("hash differs: %s vs %s", h1, h2)
	}
}

func TestFetchEscapesPlusInVL(t *testing.T) {
	raw := strings.TrimSpace(string(readTestdata(t, "rs_url_plus.txt")))
	var got string
	p := Provider{Fetch: func(_ context.Context, u string) ([]byte, error) {
		got = u
		return testdataJSON(t), nil
	}}
	ctx := provider.WithRemote(context.Background(), true)
	if _, err := p.Parse(ctx, raw); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got == "" || strings.Contains(got, "+") {
		t.Fatalf("fetched url has literal +: %s", got)
	}
}

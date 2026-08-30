package rspurs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestParseFixture(t *testing.T) {
	rawURL := testdataURL(t)
	body := testdataJSON(t)
	p := Provider{Fetch: func(_ context.Context, got string) ([]byte, error) {
		if got != rawURL {
			t.Fatalf("fetch url %s", got)
		}
		return body, nil
	}}
	receipt, err := p.Parse(context.Background(), rawURL)
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
	if receipt.ReceiptType != "sale" {
		t.Fatalf("type %s", receipt.ReceiptType)
	}
	if receipt.GrandTotal != 89.99 {
		t.Fatalf("total %v", receipt.GrandTotal)
	}
	if !receipt.IssuedAt.Equal(time.Date(2026, 8, 19, 14, 34, 52, 0, time.UTC)) {
		t.Fatalf("issued %v", receipt.IssuedAt)
	}
	if len(receipt.Items) != 1 {
		t.Fatalf("items %#v", receipt.Items)
	}
	item := receipt.Items[0]
	if item.Description != "VODA GAZIRANA KNJAZ MILOS LIMUN 1,25L (KOM)" {
		t.Fatalf("name %q", item.Description)
	}
	if item.Quantity != 1 || item.UnitPrice != 89.99 || item.TotalPrice != 89.99 {
		t.Fatalf("item %#v", item)
	}
}

func TestParseRefund(t *testing.T) {
	p := Provider{Fetch: func(context.Context, string) ([]byte, error) {
		return []byte(`{
			"invoiceRequest":{"taxId":"1","businessName":"Shop","transactionType":1},
			"invoiceResult":{"totalAmount":10,"invoiceNumber":"ABC-ABC-1","sdcTime":"2026-01-02T03:04:05Z"},
			"journal":"","isValid":true
		}`), nil
	}}
	receipt, err := p.Parse(context.Background(), testdataURL(t))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReceiptType != "refund" {
		t.Fatalf("type %s", receipt.ReceiptType)
	}
	if len(receipt.Items) != 0 {
		t.Fatalf("items %d", len(receipt.Items))
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
	if items[0].Description != "Kesa CC KESA M-UNI" || items[0].TotalPrice != 30 {
		t.Fatalf("first %#v", items[0])
	}
	if items[1].Description != "Muške cipele CCK100998-003-44" || items[1].UnitPrice != 13293 {
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

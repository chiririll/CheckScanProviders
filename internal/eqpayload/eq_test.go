package eqpayload

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseConvertsEQToMinorUnitsExactly(t *testing.T) {
	raw := `{"eq_version":"1.0.0","receipt":{"id":"r1","issued_at":"2026-08-28T18:42:00+03:00","currency":"RUB",` +
		`"receipt_type":"sale","merchant":{"name":"Пятёрочка","tax_id":"7707"},` +
		`"items":[{"description":"Сыр","quantity":0.523,"unit_price":599.99,"total_price":313.79}],` +
		`"totals":{"grand_total":1247.1}}}`
	got, err := New().Parse(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scale != 2 || got.Total != 124710 || got.Items[0].Price != 59999 || got.Items[0].Sum != 31379 {
		t.Fatalf("amounts %+v", got)
	}
	if got.Items[0].Name != "Сыр" || got.Items[0].Quantity != 0.523 || got.MerchantName != "Пятёрочка" || got.TaxID != "7707" {
		t.Fatalf("fields %+v", got)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"format":"checkscan.receipt"`) || strings.Contains(string(out), "eq_version") {
		t.Fatalf("output must be the CheckScan format: %s", out)
	}
}

func TestParseAcceptsLegacyTotalFields(t *testing.T) {
	raw := `{"receipt":{"id":"r2","currency":"RSD","totals":{"total":99.5},"items":[]}}`
	got, err := New().Parse(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 9950 || got.Currency != "RSD" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseUsesTheCurrencyScaleForEQ(t *testing.T) {
	raw := `{"receipt":{"id":"jp","currency":"JPY","totals":{"grand_total":1500},"items":[]}}`
	got, err := New().Parse(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scale != 0 || got.Total != 1500 {
		t.Fatalf("got scale %d total %d", got.Scale, got.Total)
	}
}

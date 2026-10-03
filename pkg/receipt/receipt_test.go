package receipt

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	in := Receipt{
		ID:           "ru-1",
		IssuedAt:     time.Date(2026, 8, 28, 15, 42, 0, 0, time.UTC),
		Currency:     "RUB",
		Scale:        2,
		Type:         "sale",
		MerchantName: "Пятёрочка",
		TaxID:        "7707083893",
		Total:        124750,
		Items:        []Item{{Name: "Молоко 1 л", Quantity: 2, Price: 8900, Sum: 17800}},
		Extensions:   map[string]any{"checkscan.qr_raw": "t=1"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"format":"checkscan.receipt"`, `"version":1`, `"scale":2`, `"total":124750`, `"price":8900`, `"sum":17800`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s in %s", want, text)
		}
	}
	var out Receipt
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Total != in.Total || out.Scale != 2 || out.Items[0].Sum != 17800 || out.MerchantName != in.MerchantName || !out.IssuedAt.Equal(in.IssuedAt) {
		t.Fatalf("round trip changed receipt: %+v", out)
	}
}

func TestKeepsZeroScale(t *testing.T) {
	raw, _ := json.Marshal(Receipt{ID: "jp", Currency: "JPY", Scale: 0, Total: 500})
	var out Receipt
	if err := json.Unmarshal(raw, &out); err != nil || out.Scale != 0 || out.Total != 500 {
		t.Fatalf("scale 0 lost: %s %+v %v", raw, out, err)
	}
}

func TestRequiresScale(t *testing.T) {
	var r Receipt
	if err := json.Unmarshal([]byte(`{"format":"checkscan.receipt","version":1,"total":100}`), &r); err != ErrFormat {
		t.Fatalf("err = %v, want ErrFormat for a receipt without scale", err)
	}
}

func TestRejectsOtherFormats(t *testing.T) {
	var r Receipt
	err := json.Unmarshal([]byte(`{"eq_version":"1.0.0","receipt":{"id":"x"}}`), &r)
	if err != ErrFormat {
		t.Fatalf("err = %v, want ErrFormat", err)
	}
}

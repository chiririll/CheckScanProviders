// Package receipt is the CheckScan receipt JSON: the only shape the library returns.
//
// Money is an integer count of 10^-Scale currency units: Scale 2 means kopecks
// or para. The provider that builds the receipt sets Scale; readers never guess
// it from the currency. Quantities stay fractional (0.523 kg).
package receipt

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	Format  = "checkscan.receipt"
	Version = 1
)

// ErrFormat reports JSON that is not a CheckScan receipt.
var ErrFormat = errors.New("not_a_checkscan_receipt")

type Item struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	// Price is one unit's price in minor units.
	Price int64 `json:"price"`
	// Sum is the line total in minor units.
	Sum int64 `json:"sum"`
}

type Receipt struct {
	ID       string
	IssuedAt time.Time
	Currency string
	// Scale is how many decimal digits Total, Price and Sum carry.
	Scale        int
	Type         string
	MerchantName string
	TaxID        string
	// Total is the receipt total in minor units.
	Total      int64
	Items      []Item
	Extensions map[string]any
}

type wireReceipt struct {
	Format     string         `json:"format"`
	Version    int            `json:"version"`
	ID         string         `json:"id"`
	IssuedAt   string         `json:"issued_at"`
	Currency   string         `json:"currency"`
	Scale      *int           `json:"scale"`
	Type       string         `json:"type"`
	Merchant   wireMerchant   `json:"merchant"`
	Total      int64          `json:"total"`
	Items      []Item         `json:"items"`
	Extensions map[string]any `json:"extensions"`
}

type wireMerchant struct {
	Name  string `json:"name,omitempty"`
	TaxID string `json:"tax_id,omitempty"`
}

func (r Receipt) MarshalJSON() ([]byte, error) {
	issued := r.IssuedAt
	if issued.IsZero() {
		issued = time.Now()
	}
	items := r.Items
	if items == nil {
		items = []Item{}
	}
	ext := r.Extensions
	if ext == nil {
		ext = map[string]any{}
	}
	return json.Marshal(wireReceipt{
		Format:     Format,
		Version:    Version,
		ID:         r.ID,
		IssuedAt:   issued.UTC().Format(time.RFC3339),
		Currency:   defaultString(r.Currency, "RUB"),
		Scale:      &r.Scale,
		Type:       defaultString(r.Type, "sale"),
		Merchant:   wireMerchant{Name: r.MerchantName, TaxID: r.TaxID},
		Total:      r.Total,
		Items:      items,
		Extensions: ext,
	})
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
	var wr wireReceipt
	if err := json.Unmarshal(data, &wr); err != nil {
		return err
	}
	if wr.Format != Format || wr.Scale == nil {
		return ErrFormat
	}
	issued, err := time.Parse(time.RFC3339, wr.IssuedAt)
	if err != nil {
		issued = time.Now()
	}
	items := wr.Items
	if items == nil {
		items = []Item{}
	}
	ext := wr.Extensions
	if ext == nil {
		ext = map[string]any{}
	}
	*r = Receipt{
		ID:           wr.ID,
		IssuedAt:     issued,
		Currency:     defaultString(wr.Currency, "RUB"),
		Scale:        *wr.Scale,
		Type:         defaultString(wr.Type, "sale"),
		MerchantName: wr.Merchant.Name,
		TaxID:        wr.Merchant.TaxID,
		Total:        wr.Total,
		Items:        items,
		Extensions:   ext,
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

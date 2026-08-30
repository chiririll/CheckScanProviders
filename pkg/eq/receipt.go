package eq

import (
	"encoding/json"
	"time"
)

const Version = "1.0.0"

type Item struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

type Receipt struct {
	ID           string         `json:"-"`
	IssuedAt     time.Time      `json:"-"`
	Currency     string         `json:"-"`
	ReceiptType  string         `json:"-"`
	MerchantName string         `json:"-"`
	TaxID        string         `json:"-"`
	Items        []Item         `json:"-"`
	GrandTotal   float64        `json:"-"`
	Extensions   map[string]any `json:"-"`
}

type wireEnvelope struct {
	EqVersion string          `json:"eq_version"`
	Receipt   json.RawMessage `json:"receipt"`
}

type wireReceipt struct {
	ID          string         `json:"id"`
	IssuedAt    string         `json:"issued_at"`
	Currency    string         `json:"currency"`
	ReceiptType string         `json:"receipt_type"`
	Merchant    wireMerchant   `json:"merchant"`
	Items       []Item         `json:"items"`
	Totals      wireTotals     `json:"totals"`
	GrandTotal  float64        `json:"grand_total"`
	Extensions  map[string]any `json:"extensions"`
}

type wireMerchant struct {
	Name  string `json:"name,omitempty"`
	TaxID string `json:"tax_id,omitempty"`
}

type wireTotals struct {
	GrandTotal float64 `json:"grand_total"`
	Total      float64 `json:"total"`
}

func (r Receipt) innerMap() map[string]any {
	issued := r.IssuedAt.UTC().Format(time.RFC3339)
	if r.IssuedAt.IsZero() {
		issued = time.Now().UTC().Format(time.RFC3339)
	}
	items := r.Items
	if items == nil {
		items = []Item{}
	}
	ext := r.Extensions
	if ext == nil {
		ext = map[string]any{}
	}
	return map[string]any{
		"id":           r.ID,
		"issued_at":    issued,
		"currency":     defaultString(r.Currency, "RUB"),
		"receipt_type": defaultString(r.ReceiptType, "sale"),
		"merchant":     wireMerchant{Name: r.MerchantName, TaxID: r.TaxID},
		"items":        items,
		"totals":       wireTotals{GrandTotal: r.GrandTotal},
		"extensions":   ext,
	}
}

func (r Receipt) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.innerMap())
}

func (r Receipt) Encode() ([]byte, error) {
	return json.Marshal(map[string]any{
		"eq_version": Version,
		"receipt":    r.innerMap(),
	})
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
	var env wireEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	raw := data
	if len(env.Receipt) > 0 {
		raw = env.Receipt
	}
	var wr wireReceipt
	if err := json.Unmarshal(raw, &wr); err != nil {
		return err
	}
	issued := time.Now()
	if wr.IssuedAt != "" {
		if t, err := time.Parse(time.RFC3339, wr.IssuedAt); err == nil {
			issued = t
		} else if t, err := time.Parse(time.RFC3339Nano, wr.IssuedAt); err == nil {
			issued = t
		}
	}
	total := wr.Totals.GrandTotal
	if total == 0 {
		total = wr.Totals.Total
	}
	if total == 0 {
		total = wr.GrandTotal
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
		ReceiptType:  defaultString(wr.ReceiptType, "sale"),
		MerchantName: wr.Merchant.Name,
		TaxID:        wr.Merchant.TaxID,
		Items:        items,
		GrandTotal:   total,
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

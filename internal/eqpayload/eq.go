package eqpayload

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/chiririll/CheckScanProviders/pkg/money"
	"github.com/chiririll/CheckScanProviders/pkg/receipt"
)

// eQ 1.0 is accepted only as QR input and converted at once.
type eqEnvelope struct {
	EqVersion string          `json:"eq_version"`
	Receipt   json.RawMessage `json:"receipt"`
}

type eqReceipt struct {
	ID          string         `json:"id"`
	IssuedAt    string         `json:"issued_at"`
	Currency    string         `json:"currency"`
	ReceiptType string         `json:"receipt_type"`
	Merchant    eqMerchant     `json:"merchant"`
	Items       []eqItem       `json:"items"`
	Totals      eqTotals       `json:"totals"`
	GrandTotal  json.Number    `json:"grand_total"`
	Extensions  map[string]any `json:"extensions"`
}

type eqMerchant struct {
	Name  string `json:"name"`
	TaxID string `json:"tax_id"`
}

type eqTotals struct {
	GrandTotal json.Number `json:"grand_total"`
	Total      json.Number `json:"total"`
}

type eqItem struct {
	Description string      `json:"description"`
	Quantity    json.Number `json:"quantity"`
	UnitPrice   json.Number `json:"unit_price"`
	TotalPrice  json.Number `json:"total_price"`
}

// decodeEQ reads an eQ envelope (or a bare eQ receipt) into a CheckScan receipt.
func decodeEQ(data []byte) (*receipt.Receipt, error) {
	var env eqEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	raw := data
	if len(env.Receipt) > 0 {
		raw = env.Receipt
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var wr eqReceipt
	if err := dec.Decode(&wr); err != nil {
		return nil, err
	}
	currency := wr.Currency
	if currency == "" {
		currency = "RUB"
	}
	exp := money.Exponent(currency)
	issued := time.Now()
	if t, err := time.Parse(time.RFC3339, wr.IssuedAt); err == nil {
		issued = t
	}
	items := make([]receipt.Item, 0, len(wr.Items))
	for _, item := range wr.Items {
		quantity, _ := item.Quantity.Float64()
		items = append(items, receipt.Item{
			Name:     item.Description,
			Quantity: quantity,
			Price:    minor(item.UnitPrice, exp),
			Sum:      minor(item.TotalPrice, exp),
		})
	}
	total := minor(wr.Totals.GrandTotal, exp)
	if total == 0 {
		total = minor(wr.Totals.Total, exp)
	}
	if total == 0 {
		total = minor(wr.GrandTotal, exp)
	}
	return &receipt.Receipt{
		ID:           wr.ID,
		IssuedAt:     issued,
		Currency:     currency,
		Type:         wr.ReceiptType,
		MerchantName: wr.Merchant.Name,
		TaxID:        wr.Merchant.TaxID,
		Total:        total,
		Items:        items,
		Extensions:   wr.Extensions,
	}, nil
}

func minor(n json.Number, exp int) int64 {
	if n == "" {
		return 0
	}
	value, err := money.ParseDecimal(n.String(), exp)
	if err != nil {
		return 0
	}
	return value
}

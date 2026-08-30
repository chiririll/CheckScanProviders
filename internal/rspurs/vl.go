package rspurs

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

const minVLHeader = 43

type vlPayload struct {
	RequestedBy     string
	SignedBy        string
	TotalCounter    int32
	TxCounter       int32
	TotalAmount     float64
	IssuedAt        time.Time
	InvoiceType     int
	TransactionType int
}

func decodeVL(vl string) (*vlPayload, error) {
	raw, err := base64.StdEncoding.DecodeString(vl)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(padBase64(vl))
		if err != nil {
			return nil, errors.New("invalid_vl")
		}
	}
	if len(raw) < minVLHeader {
		return nil, errors.New("invalid_vl")
	}
	amount := float64(binary.LittleEndian.Uint64(raw[25:33])) / 10000
	ms := binary.BigEndian.Uint64(raw[33:41])
	return &vlPayload{
		RequestedBy:     ascii8(raw[1:9]),
		SignedBy:        ascii8(raw[9:17]),
		TotalCounter:    int32(binary.LittleEndian.Uint32(raw[17:21])),
		TxCounter:       int32(binary.LittleEndian.Uint32(raw[21:25])),
		TotalAmount:     amount,
		IssuedAt:        time.UnixMilli(int64(ms)).UTC(),
		InvoiceType:     int(raw[41]),
		TransactionType: int(raw[42]),
	}, nil
}

func receiptFromVL(rawQR, vl string) (*eq.Receipt, error) {
	p, err := decodeVL(vl)
	if err != nil {
		return nil, err
	}
	receiptType := "sale"
	if p.TransactionType == 1 {
		receiptType = "refund"
	}
	number := p.RequestedBy + "-" + p.SignedBy + "-" + strconv.FormatInt(int64(p.TotalCounter), 10)
	issued := p.IssuedAt
	if issued.IsZero() {
		issued = time.Now().UTC()
	}
	return &eq.Receipt{
		ID:          "rs-" + number,
		IssuedAt:    issued,
		Currency:    "RSD",
		ReceiptType: receiptType,
		Items:       []eq.Item{},
		GrandTotal:  p.TotalAmount,
		Extensions: map[string]any{
			"checkscan.qr_raw": rawQR,
			"rs_purs": map[string]any{
				"invoice_number":   number,
				"requested_by":     p.RequestedBy,
				"signed_by":        p.SignedBy,
				"total_counter":    p.TotalCounter,
				"tx_counter":       p.TxCounter,
				"invoice_type":     p.InvoiceType,
				"transaction_type": p.TransactionType,
				"from_vl":          true,
			},
		},
	}, nil
}

func ascii8(b []byte) string {
	return strings.TrimSpace(strings.TrimRight(string(b), "\x00"))
}

func padBase64(s string) string {
	if m := len(s) % 4; m != 0 {
		return s + strings.Repeat("=", 4-m)
	}
	return s
}

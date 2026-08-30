package rufns

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
)

const ID = "ru_fns"

var queryRE = regexp.MustCompile(`(?i)(?:^|[?&])([a-z]+)=([^&]*)`)

type Provider struct{}

type fields struct {
	fn       string
	fd       string
	fp       string
	t        string
	n        string
	sum      *float64
	issuedAt *time.Time
}

func New() provider.Provider {
	return Provider{}
}

func (Provider) ID() string { return ID }

func (p Provider) CanHandle(rawQR string) (string, bool) {
	f := parseFields(rawQR)
	if f == nil {
		return "", false
	}
	return f.fn + "|" + f.fd + "|" + f.fp, true
}

func (p Provider) Parse(_ context.Context, rawQR string) (*eq.Receipt, error) {
	f := parseFields(rawQR)
	if f == nil {
		return nil, errors.New("invalid_qr")
	}
	issued := time.Now()
	if f.issuedAt != nil {
		issued = *f.issuedAt
	}
	receiptType := "sale"
	if f.n == "2" {
		receiptType = "refund"
	}
	total := 0.0
	if f.sum != nil {
		total = *f.sum
	}
	ru := map[string]any{
		"fn": f.fn,
		"i":  f.fd,
		"fp": f.fp,
		"t":  f.t,
		"n":  f.n,
	}
	if f.sum != nil {
		ru["s"] = *f.sum
	}
	return &eq.Receipt{
		ID:          "ru-" + f.fn + "-" + f.fd + "-" + f.fp,
		IssuedAt:    issued,
		Currency:    "RUB",
		ReceiptType: receiptType,
		Items:       []eq.Item{},
		GrandTotal:  total,
		Extensions: map[string]any{
			"checkscan.qr_raw": rawQR,
			"ru_fns":           ru,
		},
	}, nil
}

func parseFields(rawQR string) *fields {
	values := map[string]string{}
	for _, match := range queryRE.FindAllStringSubmatch(rawQR, -1) {
		key := strings.ToLower(match[1])
		decoded, err := url.QueryUnescape(match[2])
		if err != nil {
			decoded = match[2]
		}
		values[key] = decoded
	}
	fn, fd, fp := values["fn"], values["i"], values["fp"]
	if fn == "" || fd == "" || fp == "" {
		return nil
	}
	var sum *float64
	if raw, ok := values["s"]; ok && raw != "" {
		if n, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64); err == nil {
			sum = &n
		}
	}
	return &fields{
		fn:       fn,
		fd:       fd,
		fp:       fp,
		t:        values["t"],
		n:        values["n"],
		sum:      sum,
		issuedAt: parseTime(values["t"]),
	}
}

func parseTime(raw string) *time.Time {
	if len(raw) < 13 {
		return nil
	}
	compact := strings.ReplaceAll(raw, "T", "")
	if len(compact) < 12 {
		return nil
	}
	year, err1 := strconv.Atoi(compact[0:4])
	month, err2 := strconv.Atoi(compact[4:6])
	day, err3 := strconv.Atoi(compact[6:8])
	hour, err4 := strconv.Atoi(compact[8:10])
	minute, err5 := strconv.Atoi(compact[10:12])
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return nil
	}
	second := 0
	if len(compact) >= 14 {
		if n, err := strconv.Atoi(compact[12:14]); err == nil {
			second = n
		}
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
	return &t
}

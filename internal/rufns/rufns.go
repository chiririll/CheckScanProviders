package rufns

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
)

const ID = "ru_fns"

var queryRE = regexp.MustCompile(`(?i)(?:^|[?&])([a-z]+)=([^&]*)`)

// APIToken is injected at app build via -ldflags -X.
var APIToken string

type Fetcher func(ctx context.Context, qrraw string) ([]byte, error)

type Provider struct {
	Token string
	Fetch Fetcher
}

type fields struct {
	fn       string
	fd       string
	fp       string
	t        string
	n        string
	s        string
	sum      *float64
	issuedAt *time.Time
}

func New() provider.Provider {
	return Provider{}
}

func (Provider) ID() string    { return ID }
func (Provider) Label() string { return "RU" }

func (p Provider) CanHandle(rawQR string) (string, bool) {
	f := parseFields(rawQR)
	if f == nil {
		return "", false
	}
	return f.fn + "|" + f.fd + "|" + f.fp, true
}

func (p Provider) Parse(ctx context.Context, rawQR string) (*eq.Receipt, error) {
	f := parseFields(rawQR)
	if f == nil {
		nativelog.Warn("%s rufns invalid_qr", nativelog.Call(ctx))
		return nil, errors.New("invalid_qr")
	}
	receipt := receiptFromFields(rawQR, f)
	if !provider.Remote(ctx) || p.token() == "" {
		nativelog.Info("%s rufns local-only remote=%v token=%v", nativelog.Call(ctx), provider.Remote(ctx), p.token() != "")
		return receipt, nil
	}
	qrraw := f.qrraw()
	nativelog.Info("%s rufns fetch start", nativelog.Call(ctx))
	body, err := p.fetch(ctx, qrraw)
	if err != nil {
		if httplimit.IsLimit(err) {
			nativelog.Warn("%s rufns fetch limited: %v", nativelog.Call(ctx), err)
			httplimit.Mark(receipt)
		} else {
			nativelog.Warn("%s rufns fetch err: %v", nativelog.Call(ctx), err)
		}
		return receipt, nil
	}
	nativelog.Info("%s rufns fetch ok bytes=%d", nativelog.Call(ctx), len(body))
	ticket, err := parseTicket(body)
	if err != nil {
		nativelog.Warn("%s rufns ticket: %v body=%s", nativelog.Call(ctx), err, nativelog.Preview(string(body), 160))
		return receipt, nil
	}
	applyTicket(receipt, ticket)
	nativelog.Info("%s rufns ticket ok items=%d merchant=%q total=%g",
		nativelog.Call(ctx), len(receipt.Items), receipt.MerchantName, receipt.GrandTotal)
	return receipt, nil
}

func receiptFromFields(rawQR string, f *fields) *eq.Receipt {
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
	}
}

func (p Provider) token() string {
	if p.Token != "" {
		return p.Token
	}
	return strings.TrimSpace(APIToken)
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
		s:        values["s"],
		sum:      sum,
		issuedAt: parseTime(values["t"]),
	}
}

func (f *fields) qrraw() string {
	parts := []string{
		"t=" + url.QueryEscape(f.t),
		"s=" + url.QueryEscape(f.s),
		"fn=" + url.QueryEscape(f.fn),
		"i=" + url.QueryEscape(f.fd),
		"fp=" + url.QueryEscape(f.fp),
		"n=" + url.QueryEscape(f.n),
	}
	return strings.Join(parts, "&")
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

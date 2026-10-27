package rspurs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/pkg/money"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
	"github.com/chiririll/CheckScanProviders/pkg/receipt"
)

const ID = "rs_purs"

// Every PURS receipt is in dinars, counted in para.
const (
	currency = "RSD"
	scale    = 2
)

const (
	hostSUF     = "suf.purs.gov.rs"
	maxBodySize = 2 << 20
	userAgent   = "CheckScan/0.1"
)

var urlRE = regexp.MustCompile(`(?i)https?://(?:www\.)?suf\.purs\.gov\.rs/[^\s<>"']+`)

type Fetcher func(ctx context.Context, rawURL string) ([]byte, error)

type Provider struct {
	Fetch Fetcher
}

type invoiceJSON struct {
	InvoiceRequest invoiceRequest `json:"invoiceRequest"`
	InvoiceResult  invoiceResult  `json:"invoiceResult"`
	Journal        string         `json:"journal"`
	IsValid        bool           `json:"isValid"`
}

type invoiceRequest struct {
	TaxID              string    `json:"taxId"`
	BusinessName       string    `json:"businessName"`
	LocationName       string    `json:"locationName"`
	Address            string    `json:"address"`
	City               string    `json:"city"`
	AdministrativeUnit string    `json:"administrativeUnit"`
	Buyer              *string   `json:"buyer"`
	Cashier            string    `json:"cashier"`
	RequestedBy        string    `json:"requestedBy"`
	InvoiceType        int       `json:"invoiceType"`
	TransactionType    int       `json:"transactionType"`
	Payments           []payment `json:"payments"`
}

type payment struct {
	PaymentType int         `json:"paymentType"`
	Amount      json.Number `json:"amount"`
}

type invoiceResult struct {
	TotalAmount             json.Number `json:"totalAmount"`
	TransactionTypeCounter  int         `json:"transactionTypeCounter"`
	TotalCounter            int         `json:"totalCounter"`
	InvoiceCounterExtension string      `json:"invoiceCounterExtension"`
	InvoiceNumber           string      `json:"invoiceNumber"`
	SignedBy                string      `json:"signedBy"`
	SDCTime                 string      `json:"sdcTime"`
}

func New() provider.Provider {
	return Provider{}
}

func (Provider) ID() string    { return ID }
func (Provider) Label() string { return "RS" }

func (p Provider) CanHandle(rawQR string) (string, bool) {
	vl, ok := extractVL(rawQR)
	if !ok {
		return "", false
	}
	return hashVL(vl), true
}

func (p Provider) Parse(ctx context.Context, rawQR string) (*receipt.Receipt, error) {
	vl, ok := extractVL(rawQR)
	if !ok {
		nativelog.Warn("%s rspurs invalid_qr", nativelog.Call(ctx))
		return nil, errors.New("invalid_qr")
	}
	local, err := receiptFromVL(rawQR, vl)
	if !provider.Remote(ctx) {
		nativelog.Info("%s rspurs local-only vl_ok=%v", nativelog.Call(ctx), err == nil)
		return local, err
	}
	nativelog.Info("%s rspurs fetch start", nativelog.Call(ctx))
	remote, rerr := p.parseRemote(ctx, rawQR)
	if rerr == nil {
		if len(remote.Items) == 0 {
			outcome.MarkNoItems(remote)
		}
		nativelog.Info("%s rspurs remote ok items=%d merchant=%q total=%d",
			nativelog.Call(ctx), len(remote.Items), remote.MerchantName, remote.Total)
		return remote, nil
	}
	if err == nil {
		if httplimit.IsLimit(rerr) {
			nativelog.Warn("%s rspurs remote limited, fallback vl: %v", nativelog.Call(ctx), rerr)
			httplimit.Mark(local)
		} else if outcome.IsNoItems(rerr) {
			nativelog.Warn("%s rspurs remote no items, fallback vl: %v", nativelog.Call(ctx), rerr)
			outcome.MarkNoItems(local)
		} else {
			nativelog.Warn("%s rspurs remote err, fallback vl: %v", nativelog.Call(ctx), rerr)
			outcome.MarkUnavailable(local)
		}
		return local, nil
	}
	nativelog.Error("%s rspurs remote and vl failed: %v / %v", nativelog.Call(ctx), rerr, err)
	return nil, rerr
}

func (p Provider) parseRemote(ctx context.Context, rawQR string) (*receipt.Receipt, error) {
	pageURL, ok := extractURL(rawQR)
	if !ok {
		nativelog.Warn("%s rspurs remote invalid_url", nativelog.Call(ctx))
		return nil, errors.New("invalid_qr")
	}
	body, err := p.fetch(ctx, escapePlus(pageURL))
	if err != nil {
		return nil, err
	}
	receipt, err := receiptFromJSON(rawQR, body)
	if err != nil {
		nativelog.Warn("%s rspurs json: %v body=%s", nativelog.Call(ctx), err, nativelog.Preview(string(body), 160))
	}
	return receipt, err
}

func receiptFromJSON(rawQR string, body []byte) (*receipt.Receipt, error) {
	var inv invoiceJSON
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, errors.New("invalid_json")
	}
	if inv.InvoiceResult.InvoiceNumber == "" {
		return nil, errors.New("invalid_receipt")
	}
	issued := time.Now()
	if t, err := time.Parse(time.RFC3339, inv.InvoiceResult.SDCTime); err == nil {
		issued = t
	} else if t, err := time.Parse(time.RFC3339Nano, inv.InvoiceResult.SDCTime); err == nil {
		issued = t
	}
	receiptType := "sale"
	if inv.InvoiceRequest.TransactionType == 1 {
		receiptType = "refund"
	}
	rs := map[string]any{
		"invoice_number":            inv.InvoiceResult.InvoiceNumber,
		"tin":                       inv.InvoiceRequest.TaxID,
		"location_name":             strings.TrimSpace(inv.InvoiceRequest.LocationName),
		"address":                   strings.TrimSpace(inv.InvoiceRequest.Address),
		"city":                      strings.TrimSpace(inv.InvoiceRequest.City),
		"administrative_unit":       strings.TrimSpace(inv.InvoiceRequest.AdministrativeUnit),
		"cashier":                   inv.InvoiceRequest.Cashier,
		"requested_by":              inv.InvoiceRequest.RequestedBy,
		"signed_by":                 inv.InvoiceResult.SignedBy,
		"invoice_type":              inv.InvoiceRequest.InvoiceType,
		"transaction_type":          inv.InvoiceRequest.TransactionType,
		"invoice_counter_extension": inv.InvoiceResult.InvoiceCounterExtension,
		"is_valid":                  inv.IsValid,
	}
	if inv.InvoiceRequest.Buyer != nil && *inv.InvoiceRequest.Buyer != "" {
		rs["buyer"] = *inv.InvoiceRequest.Buyer
	}
	if len(inv.InvoiceRequest.Payments) > 0 {
		pays := make([]map[string]any, 0, len(inv.InvoiceRequest.Payments))
		for _, pay := range inv.InvoiceRequest.Payments {
			pays = append(pays, map[string]any{
				"payment_type": pay.PaymentType,
				"amount":       amountPara(pay.Amount),
			})
		}
		rs["payments"] = pays
	}
	return &receipt.Receipt{
		ID:           "rs-" + inv.InvoiceResult.InvoiceNumber,
		IssuedAt:     issued,
		Currency:     currency,
		Scale:        scale,
		Type:         receiptType,
		MerchantName: strings.TrimSpace(inv.InvoiceRequest.BusinessName),
		TaxID:        inv.InvoiceRequest.TaxID,
		Items:        parseJournalItems(inv.Journal),
		Total:        amountPara(inv.InvoiceResult.TotalAmount),
		Extensions: map[string]any{
			"checkscan.qr_raw": rawQR,
			"rs_purs":          rs,
		},
	}, nil
}

func (p Provider) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if err := httplimit.Acquire(ctx, hostSUF, provider.Wait(ctx)); err != nil {
		return nil, err
	}
	if p.Fetch != nil {
		body, err := p.Fetch(ctx, rawURL)
		if err != nil {
			httplimit.NoteError(hostSUF, err)
		}
		return body, err
	}
	return defaultFetch(ctx, rawURL)
}

func defaultFetch(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		nativelog.Warn("%s rspurs http GET %s: %v", nativelog.Call(ctx), hostSUF, err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		nativelog.Warn("%s rspurs http GET %s status=%d", nativelog.Call(ctx), hostSUF, resp.StatusCode)
		httplimit.Note(hostSUF, resp.StatusCode, resp.Header)
		return nil, fmt.Errorf("http_%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		nativelog.Warn("%s rspurs http read: %v", nativelog.Call(ctx), err)
		return nil, err
	}
	nativelog.Info("%s rspurs http GET %s status=%d bytes=%d", nativelog.Call(ctx), hostSUF, resp.StatusCode, len(body))
	return body, nil
}

func extractURL(rawQR string) (string, bool) {
	raw := strings.TrimSpace(rawQR)
	if found := urlRE.FindString(raw); found != "" {
		raw = strings.TrimRight(found, ".,);]")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if strings.ToLower(u.Hostname()) != hostSUF {
		return "", false
	}
	if vlParam(u) == "" {
		return "", false
	}
	return raw, true
}

// escapePlus sends a literal "+" in the query as %2B; the server would otherwise
// decode it as a space and see a corrupted vl.
func escapePlus(rawURL string) string {
	i := strings.Index(rawURL, "?")
	if i < 0 {
		return rawURL
	}
	return rawURL[:i+1] + strings.ReplaceAll(rawURL[i+1:], "+", "%2B")
}

func extractVL(rawQR string) (string, bool) {
	pageURL, ok := extractURL(rawQR)
	if !ok {
		return "", false
	}
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", false
	}
	vl := vlParam(u)
	if vl == "" {
		return "", false
	}
	return vl, true
}

// vlParam returns the "vl" query value. SUF puts base64 in it unescaped, so a
// literal "+" is escaped first; otherwise query decoding would turn it into a space.
func vlParam(u *url.URL) string {
	q, _ := url.ParseQuery(strings.ReplaceAll(u.RawQuery, "+", "%2B"))
	return q.Get("vl")
}

func hashVL(vl string) string {
	sum := sha256.Sum256([]byte(vl))
	return hex.EncodeToString(sum[:])
}

// amountPara reads a PURS JSON amount (dinars) into para without going through float.
func amountPara(n json.Number) int64 {
	if n == "" {
		return 0
	}
	value, err := money.ParseDecimal(n.String(), scale)
	if err != nil {
		return 0
	}
	return value
}

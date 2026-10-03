package rufns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/pkg/money"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
	"github.com/chiririll/CheckScanProviders/pkg/receipt"
)

const apiHost = "proverkacheka.com"

const (
	apiURL      = "https://proverkacheka.com/api/v1/check/get"
	maxBodySize = 2 << 20
	userAgent   = "CheckScan/0.1"
)

type apiResponse struct {
	Code int `json:"code"`
	Data struct {
		JSON json.RawMessage `json:"json"`
	} `json:"data"`
}

type ticket struct {
	User        string       `json:"user"`
	UserInn     string       `json:"userInn"`
	RetailPlace string       `json:"retailPlace"`
	DateTime    string       `json:"dateTime"`
	TotalSum    json.Number  `json:"totalSum"`
	Operation   int          `json:"operationType"`
	Items       []ticketItem `json:"items"`
}

type ticketItem struct {
	Name     string      `json:"name"`
	Price    json.Number `json:"price"`
	Quantity float64     `json:"quantity"`
	Sum      json.Number `json:"sum"`
}

func (p Provider) fetch(ctx context.Context, qrraw string) ([]byte, error) {
	if err := httplimit.Acquire(ctx, apiHost, provider.Wait(ctx)); err != nil {
		return nil, err
	}
	if p.Fetch != nil {
		body, err := p.Fetch(ctx, qrraw)
		if err != nil {
			httplimit.NoteError(apiHost, err)
		}
		return body, err
	}
	return defaultFetch(ctx, p.token(), qrraw)
}

func defaultFetch(ctx context.Context, token, qrraw string) ([]byte, error) {
	form := url.Values{}
	form.Set("token", token)
	form.Set("qrraw", qrraw)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cookie", "ENGID=1.1")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		nativelog.Warn("%s rufns http POST %s: %v", nativelog.Call(ctx), apiHost, err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		nativelog.Warn("%s rufns http POST %s status=%d", nativelog.Call(ctx), apiHost, resp.StatusCode)
		httplimit.Note(apiHost, resp.StatusCode, resp.Header)
		return nil, fmt.Errorf("http_%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		nativelog.Warn("%s rufns http read: %v", nativelog.Call(ctx), err)
		return nil, err
	}
	nativelog.Info("%s rufns http POST %s status=%d bytes=%d", nativelog.Call(ctx), apiHost, resp.StatusCode, len(body))
	return body, nil
}

func parseTicket(body []byte) (*ticket, map[string]any, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}
	if resp.Code != 1 || len(resp.Data.JSON) == 0 || string(resp.Data.JSON) == "null" {
		return nil, nil, errors.New("check_unavailable")
	}
	var t ticket
	if err := json.Unmarshal(resp.Data.JSON, &t); err != nil {
		return nil, nil, err
	}
	// Numbers stay json.Number so raw provider amounts are echoed exactly.
	var extra map[string]any
	dec := json.NewDecoder(bytes.NewReader(resp.Data.JSON))
	dec.UseNumber()
	if err := dec.Decode(&extra); err != nil {
		return nil, nil, err
	}
	delete(extra, "items")
	return &t, extra, nil
}

// applyTicket fills r from the FNS ticket. FNS amounts are integer kopecks.
func applyTicket(r *receipt.Receipt, t *ticket, extra map[string]any) {
	if name := strings.TrimSpace(t.RetailPlace); name != "" {
		r.MerchantName = name
	} else if name := strings.TrimSpace(t.User); name != "" {
		r.MerchantName = name
	}
	if inn := strings.TrimSpace(t.UserInn); inn != "" {
		r.TaxID = inn
	}
	if total := kopecks(t.TotalSum); total > 0 {
		r.Total = total
	}
	if t.Operation == 2 {
		r.Type = "refund"
	}
	if issued, ok := parseTicketTime(t.DateTime); ok {
		r.IssuedAt = issued
	}
	items := make([]receipt.Item, 0, len(t.Items))
	for _, item := range t.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		items = append(items, receipt.Item{
			Name:     name,
			Quantity: item.Quantity,
			Price:    kopecks(item.Price),
			Sum:      kopecks(item.Sum),
		})
	}
	if len(items) > 0 {
		r.Items = items
	}
	if extra != nil {
		if r.Extensions == nil {
			r.Extensions = map[string]any{}
		}
		r.Extensions["ru_fns"] = extra
	}
}

func kopecks(n json.Number) int64 {
	if n == "" {
		return 0
	}
	value, err := money.ParseDecimal(n.String(), 0)
	if err != nil {
		return 0
	}
	return value
}

func parseTicketTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02T15:04:05", raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}

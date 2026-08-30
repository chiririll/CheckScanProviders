package rufns

import (
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
	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
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
		JSON *ticket `json:"json"`
	} `json:"data"`
}

type ticket struct {
	User        string       `json:"user"`
	UserInn     string       `json:"userInn"`
	RetailPlace string       `json:"retailPlace"`
	DateTime    string       `json:"dateTime"`
	TotalSum    float64      `json:"totalSum"`
	Operation   int          `json:"operationType"`
	Items       []ticketItem `json:"items"`
}

type ticketItem struct {
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Sum      float64 `json:"sum"`
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
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		httplimit.Note(apiHost, resp.StatusCode, resp.Header)
		return nil, fmt.Errorf("http_%d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
}

func parseTicket(body []byte) (*ticket, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 1 || resp.Data.JSON == nil {
		return nil, errors.New("check_unavailable")
	}
	return resp.Data.JSON, nil
}

func applyTicket(receipt *eq.Receipt, t *ticket) {
	if name := strings.TrimSpace(t.RetailPlace); name != "" {
		receipt.MerchantName = name
	} else if name := strings.TrimSpace(t.User); name != "" {
		receipt.MerchantName = name
	}
	if inn := strings.TrimSpace(t.UserInn); inn != "" {
		receipt.TaxID = inn
	}
	if t.TotalSum > 0 {
		receipt.GrandTotal = t.TotalSum / 100
	}
	if t.Operation == 2 {
		receipt.ReceiptType = "refund"
	}
	if issued, ok := parseTicketTime(t.DateTime); ok {
		receipt.IssuedAt = issued
	}
	items := make([]eq.Item, 0, len(t.Items))
	for _, item := range t.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		items = append(items, eq.Item{
			Description: name,
			Quantity:    item.Quantity,
			UnitPrice:   item.Price / 100,
			TotalPrice:  item.Sum / 100,
		})
	}
	if len(items) > 0 {
		receipt.Items = items
	}
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

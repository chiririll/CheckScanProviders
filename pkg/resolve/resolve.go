package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chiririll/CheckScanProviders/internal/eqpayload"
	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/internal/rspurs"
	"github.com/chiririll/CheckScanProviders/internal/rufns"
	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
)

var (
	ErrUnknownFormat = errors.New("unknown_format")
	ErrParse         = errors.New("parse_error")
)

type Match struct {
	AdapterID string `json:"adapter_id"`
	Hash      string `json:"hash"`
	Label     string `json:"label"`
}

type Result struct {
	AdapterID string      `json:"adapter_id"`
	Hash      string      `json:"hash"`
	Label     string      `json:"label"`
	EqVersion string      `json:"eq_version"`
	Receipt   *eq.Receipt `json:"receipt"`
}

type jsonError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func DefaultRegistry() *provider.Registry {
	return provider.NewRegistry(eqpayload.New(), rspurs.New(), rufns.New())
}

func MatchQR(rawQR, hint string, registry *provider.Registry) (Match, error) {
	if registry == nil {
		registry = DefaultRegistry()
	}
	if hint != "" {
		p, ok := registry.ByID(hint)
		if !ok {
			return Match{}, ErrUnknownFormat
		}
		hash, ok := p.CanHandle(rawQR)
		if !ok || hash == "" {
			return Match{}, ErrUnknownFormat
		}
		return matchOf(p, hash), nil
	}
	for _, p := range registry.All() {
		hash, ok := p.CanHandle(rawQR)
		if ok && hash != "" {
			return matchOf(p, hash), nil
		}
	}
	return Match{}, ErrUnknownFormat
}

func Resolve(ctx context.Context, rawQR, hint string, registry *provider.Registry, current *eq.Receipt) (Result, error) {
	if registry == nil {
		registry = DefaultRegistry()
	}
	match, err := MatchQR(rawQR, hint, registry)
	if err != nil {
		return Result{}, err
	}
	p, ok := registry.ByID(match.AdapterID)
	if !ok {
		return Result{}, ErrUnknownFormat
	}
	receipt, err := p.Parse(ctx, rawQR)
	if err != nil {
		return Result{}, err
	}
	if current != nil && !outcome.IsRicher(receipt, current) {
		nativelog.Info("%s resolve keep current incoming not richer", nativelog.Call(ctx))
		receipt = current
	}
	return Result{
		AdapterID: match.AdapterID,
		Hash:      match.Hash,
		Label:     match.Label,
		EqVersion: eq.Version,
		Receipt:   receipt,
	}, nil
}

func matchOf(p provider.Provider, hash string) Match {
	return Match{AdapterID: p.ID(), Hash: hash, Label: p.Label()}
}

func MatchJSON(rawQR, hint string) string {
	id := nativelog.Next()
	nativelog.Info("%s match start hint=%q qr=%s", id, hint, nativelog.Preview(rawQR, 96))
	match, err := MatchQR(rawQR, hint, nil)
	if err != nil {
		nativelog.Info("%s match unknown_format", id)
		return encodeError("unknown_format", err.Error())
	}
	nativelog.Info("%s match ok adapter=%s hash=%s label=%s", id, match.AdapterID, match.Hash, match.Label)
	b, err := json.Marshal(match)
	if err != nil {
		nativelog.Error("%s match encode: %v", id, err)
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func ResolveJSON(ctx context.Context, rawQR, hint, currentJSON string) string {
	id := nativelog.Next()
	ctx = nativelog.WithCall(ctx, id)
	nativelog.Info("%s resolve start remote=%v wait=%v hint=%q current=%v qr=%s",
		id, provider.Remote(ctx), provider.Wait(ctx), hint, currentJSON != "", nativelog.Preview(rawQR, 96))
	result, err := Resolve(ctx, rawQR, hint, nil, parseCurrent(currentJSON))
	if err != nil {
		code := "parse_error"
		if errors.Is(err, ErrUnknownFormat) {
			code = "unknown_format"
		}
		nativelog.Error("%s resolve %s: %v", id, code, err)
		return encodeError(code, err.Error())
	}
	nativelog.Info("%s resolve ok adapter=%s hash=%s %s", id, result.AdapterID, result.Hash, receiptSummary(result.Receipt))
	b, err := json.Marshal(result)
	if err != nil {
		nativelog.Error("%s resolve encode: %v", id, err)
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func ProvidersJSON() string {
	type item struct {
		ID      string            `json:"id"`
		Label   string            `json:"label"`
		Secrets []provider.Secret `json:"secrets,omitempty"`
	}
	list := make([]item, 0)
	for _, p := range DefaultRegistry().All() {
		entry := item{ID: p.ID(), Label: p.Label()}
		if cfg, ok := p.(provider.HasSecrets); ok {
			entry.Secrets = cfg.Secrets()
		}
		list = append(list, entry)
	}
	b, err := json.Marshal(list)
	if err != nil {
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func parseCurrent(raw string) *eq.Receipt {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var receipt eq.Receipt
	if err := json.Unmarshal([]byte(raw), &receipt); err != nil {
		return nil
	}
	return &receipt
}

func encodeError(code, message string) string {
	var out jsonError
	out.Error.Code = code
	out.Error.Message = message
	b, _ := json.Marshal(out)
	return string(b)
}

func receiptSummary(r *eq.Receipt) string {
	if r == nil {
		return "receipt=nil"
	}
	limited := false
	noItems := false
	if r.Extensions != nil {
		if v, ok := r.Extensions["checkscan.rate_limited"].(bool); ok {
			limited = v
		}
		if v, ok := r.Extensions["checkscan.items_unavailable"].(bool); ok {
			noItems = v
		}
	}
	return fmt.Sprintf("id=%s items=%d total=%g merchant=%q limited=%v no_items=%v",
		r.ID, len(r.Items), r.GrandTotal, r.MerchantName, limited, noItems)
}

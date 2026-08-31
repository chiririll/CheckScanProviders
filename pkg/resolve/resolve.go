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
	"github.com/chiririll/CheckScanProviders/pkg/status"
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

type SettingField struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Label string `json:"label"`
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
		code, message := Classify(nil, err)
		nativelog.Info("%s match status=%d", id, code)
		return encodeEnvelope(code, message, nil)
	}
	nativelog.Info("%s match ok adapter=%s hash=%s label=%s", id, match.AdapterID, match.Hash, match.Label)
	return encodeEnvelope(status.OK, "", match)
}

func ResolveJSON(ctx context.Context, rawQR, hint, currentJSON string) string {
	id := nativelog.Next()
	ctx = nativelog.WithCall(ctx, id)
	nativelog.Info("%s resolve start remote=%v wait=%v hint=%q current=%v qr=%s",
		id, provider.Remote(ctx), provider.Wait(ctx), hint, currentJSON != "", nativelog.Preview(rawQR, 96))
	result, err := Resolve(ctx, rawQR, hint, nil, parseCurrent(currentJSON))
	if err != nil {
		code, message := Classify(nil, err)
		nativelog.Error("%s resolve status=%d: %s", id, code, message)
		return encodeEnvelope(code, message, nil)
	}
	code, message := Classify(result.Receipt, nil)
	nativelog.Info("%s resolve status=%d adapter=%s hash=%s %s", id, code, result.AdapterID, result.Hash, receiptSummary(result.Receipt))
	return encodeEnvelope(code, message, result)
}

func SettingsJSON() string {
	fields := make([]SettingField, 0)
	for _, p := range DefaultRegistry().All() {
		cfg, ok := p.(provider.HasSecrets)
		if !ok {
			continue
		}
		for _, secret := range cfg.Secrets() {
			if secret.ID == "" {
				continue
			}
			fields = append(fields, SettingField{
				Key:   p.ID() + "." + secret.ID,
				Type:  "secret",
				Label: p.Label(),
			})
		}
	}
	return encodeEnvelope(status.OK, "", map[string]any{"fields": fields})
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

func receiptSummary(r *eq.Receipt) string {
	if r == nil {
		return "receipt=nil"
	}
	return fmt.Sprintf("id=%s items=%d total=%g merchant=%q limited=%v secret=%v unreachable=%v",
		r.ID, len(r.Items), r.GrandTotal, r.MerchantName,
		outcome.HasFlag(r, "checkscan.rate_limited"),
		outcome.HasFlag(r, outcome.NeedsSecretKey),
		outcome.HasFlag(r, outcome.UnavailableKey))
}

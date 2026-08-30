package resolve

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/chiririll/CheckScanProviders/internal/eqpayload"
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

func Resolve(ctx context.Context, rawQR, hint string, registry *provider.Registry) (Result, error) {
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
	match, err := MatchQR(rawQR, hint, nil)
	if err != nil {
		return encodeError("unknown_format", err.Error())
	}
	b, err := json.Marshal(match)
	if err != nil {
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func ResolveJSON(ctx context.Context, rawQR, hint string) string {
	result, err := Resolve(ctx, rawQR, hint, nil)
	if err != nil {
		code := "parse_error"
		if errors.Is(err, ErrUnknownFormat) {
			code = "unknown_format"
		}
		return encodeError(code, err.Error())
	}
	b, err := json.Marshal(result)
	if err != nil {
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func ProvidersJSON() string {
	type item struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	list := make([]item, 0)
	for _, p := range DefaultRegistry().All() {
		list = append(list, item{ID: p.ID(), Label: p.Label()})
	}
	b, err := json.Marshal(list)
	if err != nil {
		return encodeError("parse_error", err.Error())
	}
	return string(b)
}

func encodeError(code, message string) string {
	var out jsonError
	out.Error.Code = code
	out.Error.Message = message
	b, _ := json.Marshal(out)
	return string(b)
}

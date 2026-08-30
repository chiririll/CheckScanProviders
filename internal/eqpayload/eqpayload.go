package eqpayload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/provider"
)

const ID = "eq_payload"

type Provider struct{}

func New() provider.Provider {
	return Provider{}
}

func (Provider) ID() string { return ID }

func (p Provider) CanHandle(rawQR string) (string, bool) {
	text := strings.TrimSpace(rawQR)
	m, ok := asMap(text)
	if !ok {
		return "", false
	}
	if _, hasVersion := m["eq_version"]; !hasVersion {
		if _, hasReceipt := m["receipt"]; !hasReceipt {
			return "", false
		}
	}
	var receipt eq.Receipt
	if err := json.Unmarshal([]byte(text), &receipt); err != nil {
		return "", false
	}
	if receipt.ID != "" {
		return receipt.ID, true
	}
	return hashRaw(text), true
}

func (p Provider) Parse(_ context.Context, rawQR string) (*eq.Receipt, error) {
	text := strings.TrimSpace(rawQR)
	if _, ok := asMap(text); !ok {
		return nil, errors.New("invalid_json")
	}
	var receipt eq.Receipt
	if err := json.Unmarshal([]byte(text), &receipt); err != nil {
		return nil, errors.New("invalid_json")
	}
	return &receipt, nil
}

func asMap(raw string) (map[string]any, bool) {
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, false
	}
	return m, true
}

func hashRaw(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

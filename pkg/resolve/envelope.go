package resolve

import (
	"encoding/json"
	"errors"

	"github.com/chiririll/CheckScanProviders/internal/httplimit"
	"github.com/chiririll/CheckScanProviders/internal/outcome"
	"github.com/chiririll/CheckScanProviders/pkg/eq"
	"github.com/chiririll/CheckScanProviders/pkg/status"
)

type Envelope struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Classify(receipt *eq.Receipt, err error) (int, string) {
	if err != nil && receipt == nil {
		if errors.Is(err, ErrUnknownFormat) {
			return status.UnknownFormat, err.Error()
		}
		return status.ParseError, err.Error()
	}
	if receipt == nil {
		return status.ParseError, "empty_receipt"
	}
	if outcome.HasFlag(receipt, httplimit.ExtensionKey) {
		return status.RateLimited, "rate_limited"
	}
	if outcome.HasFlag(receipt, outcome.NeedsSecretKey) {
		return status.NeedsSecret, "needs_secret"
	}
	if outcome.HasFlag(receipt, outcome.UnavailableKey) {
		return status.Unavailable, "unavailable"
	}
	if len(receipt.Items) == 0 {
		return status.Incomplete, ""
	}
	return status.OK, ""
}

func encodeEnvelope(code int, message string, data any) string {
	b, err := json.Marshal(Envelope{Status: code, Message: message, Data: data})
	if err != nil {
		fallback, _ := json.Marshal(Envelope{Status: status.ParseError, Message: err.Error()})
		return string(fallback)
	}
	return string(b)
}

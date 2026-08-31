package outcome

import "github.com/chiririll/CheckScanProviders/pkg/eq"

const (
	ItemsUnavailableKey = "checkscan.items_unavailable"
	NeedsSecretKey      = "checkscan.needs_secret"
	UnavailableKey      = "checkscan.api_unreachable"
)

func MarkNoItems(receipt *eq.Receipt) {
	markFlag(receipt, ItemsUnavailableKey, receipt != nil && len(receipt.Items) == 0)
}

func MarkNeedsSecret(receipt *eq.Receipt) {
	markFlag(receipt, NeedsSecretKey, true)
}

func MarkUnavailable(receipt *eq.Receipt) {
	markFlag(receipt, UnavailableKey, true)
}

func HasFlag(receipt *eq.Receipt, key string) bool {
	if receipt == nil || receipt.Extensions == nil {
		return false
	}
	value := receipt.Extensions[key]
	return value == true || value == "true"
}

func markFlag(receipt *eq.Receipt, key string, ok bool) {
	if receipt == nil || !ok {
		return
	}
	if receipt.Extensions == nil {
		receipt.Extensions = map[string]any{}
	}
	receipt.Extensions[key] = true
}

func IsNoItems(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "http_400", "invalid_json", "invalid_receipt":
		return true
	default:
		return false
	}
}

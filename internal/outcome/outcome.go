package outcome

import "github.com/chiririll/CheckScanProviders/pkg/eq"

const ItemsUnavailableKey = "checkscan.items_unavailable"

func MarkNoItems(receipt *eq.Receipt) {
	if receipt == nil || len(receipt.Items) > 0 {
		return
	}
	if receipt.Extensions == nil {
		receipt.Extensions = map[string]any{}
	}
	receipt.Extensions[ItemsUnavailableKey] = true
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

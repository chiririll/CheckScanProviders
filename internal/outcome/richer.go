package outcome

import (
	"strings"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

func hasText(value string) bool {
	return strings.TrimSpace(value) != ""
}

// IsRicher reports whether incoming adds items, merchant, tax id, or a total
// that current does not have.
func IsRicher(incoming, current *eq.Receipt) bool {
	if incoming == nil || current == nil {
		return incoming != nil
	}
	if len(incoming.Items) > len(current.Items) {
		return true
	}
	if hasText(incoming.MerchantName) && !hasText(current.MerchantName) {
		return true
	}
	if hasText(incoming.TaxID) && !hasText(current.TaxID) {
		return true
	}
	if incoming.GrandTotal > 0 && current.GrandTotal == 0 {
		return true
	}
	return false
}

func PreferRicher(incoming, current *eq.Receipt) *eq.Receipt {
	if current == nil || IsRicher(incoming, current) {
		return incoming
	}
	return current
}

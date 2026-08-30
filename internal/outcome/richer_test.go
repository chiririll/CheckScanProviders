package outcome

import (
	"testing"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

func TestIsRicher(t *testing.T) {
	empty := &eq.Receipt{GrandTotal: 1749}
	withItems := &eq.Receipt{
		GrandTotal:   1749,
		MerchantName: "Shop",
		Items:        []eq.Item{{Description: "Voda", Quantity: 1, UnitPrice: 1749, TotalPrice: 1749}},
	}
	if !IsRicher(withItems, empty) {
		t.Fatal("items must win")
	}
	if IsRicher(empty, withItems) {
		t.Fatal("empty must not replace items")
	}
	if IsRicher(empty, empty) {
		t.Fatal("same payload is not richer")
	}
	kept := PreferRicher(empty, withItems)
	if len(kept.Items) != 1 {
		t.Fatal("prefer current")
	}
}

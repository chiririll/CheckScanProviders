package outcome

import (
	"testing"

	"github.com/chiririll/CheckScanProviders/pkg/receipt"
)

func TestIsRicher(t *testing.T) {
	empty := &receipt.Receipt{Total: 174900}
	withItems := &receipt.Receipt{
		Total:        174900,
		MerchantName: "Shop",
		Items:        []receipt.Item{{Name: "Voda", Quantity: 1, Price: 174900, Sum: 174900}},
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

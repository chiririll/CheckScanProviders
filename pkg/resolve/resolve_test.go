package resolve_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chiririll/CheckScanProviders/pkg/resolve"
)

func TestMatchFNSQueryAndURLSameHash(t *testing.T) {
	query := read(t, "fns_query.txt")
	url := read(t, "fns_url.txt")

	q, err := resolve.MatchQR(query, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := resolve.MatchQR(url, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if q.AdapterID != "ru_fns" || u.AdapterID != "ru_fns" {
		t.Fatalf("adapter: %s / %s", q.AdapterID, u.AdapterID)
	}
	want := "8710000100905518|12|4135164163"
	if q.Hash != want || u.Hash != want {
		t.Fatalf("hash: %s / %s", q.Hash, u.Hash)
	}
}

func TestMatchEQUsesReceiptID(t *testing.T) {
	raw := read(t, "eq_with_id.json")
	m, err := resolve.MatchQR(raw, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.AdapterID != "eq_payload" {
		t.Fatalf("adapter %s", m.AdapterID)
	}
	if m.Hash != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("hash %s", m.Hash)
	}
}

func TestMatchEQWithoutIDHashesRawQR(t *testing.T) {
	raw := strings.TrimSpace(read(t, "eq_without_id.json"))
	m, err := resolve.MatchQR(raw, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(raw))
	if m.Hash != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash %s", m.Hash)
	}
	again, err := resolve.MatchQR(raw, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.Hash != m.Hash {
		t.Fatal("hash must be stable")
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := resolve.MatchQR("not-a-receipt", "", nil); err != resolve.ErrUnknownFormat {
		t.Fatalf("got %v", err)
	}
	raw := resolve.MatchJSON("not-a-receipt", "")
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["code"] != "unknown_format" {
		t.Fatalf("json error: %s", raw)
	}
}

func TestResolveFNSIncomplete(t *testing.T) {
	result, err := resolve.Resolve(context.Background(), read(t, "fns_query.txt"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt == nil || len(result.Receipt.Items) != 0 {
		t.Fatal("expected no items")
	}
	if result.Receipt.GrandTotal != 1247 {
		t.Fatalf("total %v", result.Receipt.GrandTotal)
	}
	if result.Receipt.Currency != "RUB" {
		t.Fatalf("currency %s", result.Receipt.Currency)
	}
}

func TestResolveEQHasItems(t *testing.T) {
	result, err := resolve.Resolve(context.Background(), read(t, "eq_with_id.json"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.AdapterID != "eq_payload" {
		t.Fatalf("adapter %s", result.AdapterID)
	}
	if result.Receipt.MerchantName != "Пятёрочка" {
		t.Fatalf("merchant %s", result.Receipt.MerchantName)
	}
	if len(result.Receipt.Items) != 1 {
		t.Fatalf("items %d", len(result.Receipt.Items))
	}
}

func TestChainPrefersEQ(t *testing.T) {
	m, err := resolve.MatchQR(read(t, "eq_with_id.json"), "", nil)
	if err != nil || m.AdapterID != "eq_payload" {
		t.Fatalf("%v %v", m, err)
	}
}

func TestMatchRSPursURL(t *testing.T) {
	raw := strings.TrimSpace(read(t, "rs_url.txt"))
	m, err := resolve.MatchQR(raw, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.AdapterID != "rs_purs" {
		t.Fatalf("adapter %s", m.AdapterID)
	}
	if m.Hash == "" {
		t.Fatal("empty hash")
	}
	again, err := resolve.MatchQR(raw, "", nil)
	if err != nil || again.Hash != m.Hash {
		t.Fatalf("hash must be stable: %v %s", err, again.Hash)
	}
}

func TestHintForcesProvider(t *testing.T) {
	raw := read(t, "fns_query.txt")
	if _, err := resolve.MatchQR(raw, "eq_payload", nil); err != resolve.ErrUnknownFormat {
		t.Fatalf("expected unknown, got %v", err)
	}
	m, err := resolve.MatchQR(raw, "ru_fns", nil)
	if err != nil || m.AdapterID != "ru_fns" {
		t.Fatalf("%v %v", m, err)
	}
}

func TestProvidersJSON(t *testing.T) {
	var list []map[string]string
	if err := json.Unmarshal([]byte(resolve.ProvidersJSON()), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0]["id"] != "eq_payload" || list[1]["id"] != "rs_purs" || list[2]["id"] != "ru_fns" {
		t.Fatalf("%v", list)
	}
}

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

package rspurs

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/chiririll/CheckScanProviders/pkg/money"
	"github.com/chiririll/CheckScanProviders/pkg/receipt"
)

var (
	priceLineRE = regexp.MustCompile(`^\s*(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:[.,]\d+)?)\s+(\d+(?:[.,]\d+)?)\s+(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:[.,]\d+)?)\s*$`)
	vatSuffixRE = regexp.MustCompile(`\s*\(\p{L}\)\s*$`)
	sepLineRE   = regexp.MustCompile(`^[=\-]{8,}\s*$`)
)

func parseJournalItems(journal string) []receipt.Item {
	if journal == "" {
		return []receipt.Item{}
	}
	lines := splitJournalLines(journal)
	start := itemSectionStart(lines)
	if start < 0 {
		return []receipt.Item{}
	}
	var items []receipt.Item
	var nameParts []string
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if sepLineRE.MatchString(trimmed) || strings.Contains(trimmed, "Укупан износ") || strings.Contains(trimmed, "Ukupan iznos") {
			break
		}
		if m := priceLineRE.FindStringSubmatch(trimmed); m != nil {
			name := strings.TrimSpace(strings.Join(nameParts, ""))
			name = vatSuffixRE.ReplaceAllString(name, "")
			name = strings.TrimSpace(name)
			nameParts = nil
			if name == "" {
				continue
			}
			items = append(items, receipt.Item{
				Name:     name,
				Price:    parsePara(m[1]),
				Quantity: parseQuantity(m[2]),
				Sum:      parsePara(m[3]),
			})
			continue
		}
		nameParts = append(nameParts, strings.TrimRightFunc(line, unicode.IsSpace))
	}
	if items == nil {
		return []receipt.Item{}
	}
	return items
}

func itemSectionStart(lines []string) int {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Назив") && (strings.Contains(trimmed, "Укупно") || strings.Contains(trimmed, "Цена")) {
			return i + 1
		}
		if strings.Contains(trimmed, "Naziv") && (strings.Contains(trimmed, "Ukupno") || strings.Contains(trimmed, "Cena")) {
			return i + 1
		}
	}
	return -1
}

func splitJournalLines(journal string) []string {
	normalized := strings.ReplaceAll(journal, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

// journalDecimal normalizes "1.234,56" (dot thousands, comma decimals) to "1234.56".
func journalDecimal(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	return s
}

// parsePara reads a journal amount into para (1/100 dinar).
func parsePara(raw string) int64 {
	n, err := money.ParseDecimal(journalDecimal(raw), money.Exponent(currency))
	if err != nil {
		return 0
	}
	return n
}

func parseQuantity(raw string) float64 {
	n, err := strconv.ParseFloat(journalDecimal(raw), 64)
	if err != nil {
		return 0
	}
	return n
}

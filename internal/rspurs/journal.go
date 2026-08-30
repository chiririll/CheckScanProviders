package rspurs

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

var (
	priceLineRE = regexp.MustCompile(`^\s*(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:[.,]\d+)?)\s+(\d+(?:[.,]\d+)?)\s+(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:[.,]\d+)?)\s*$`)
	vatSuffixRE = regexp.MustCompile(`\s*\(\p{L}\)\s*$`)
	sepLineRE   = regexp.MustCompile(`^[=\-]{8,}\s*$`)
)

func parseJournalItems(journal string) []eq.Item {
	if journal == "" {
		return []eq.Item{}
	}
	lines := splitJournalLines(journal)
	start := itemSectionStart(lines)
	if start < 0 {
		return []eq.Item{}
	}
	var items []eq.Item
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
			items = append(items, eq.Item{
				Description: name,
				UnitPrice:   parseDecimal(m[1]),
				Quantity:    parseDecimal(m[2]),
				TotalPrice:  parseDecimal(m[3]),
			})
			continue
		}
		nameParts = append(nameParts, strings.TrimRightFunc(line, unicode.IsSpace))
	}
	if items == nil {
		return []eq.Item{}
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

func parseDecimal(raw string) float64 {
	s := strings.TrimSpace(raw)
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return n
}

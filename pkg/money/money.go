// Package money converts amounts to integer minor units (kopecks, para) without floats.
package money

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalid reports a string that is not a plain decimal number.
var ErrInvalid = errors.New("invalid_amount")

// minorDigits is the ISO 4217 exponent for currencies whose exponent is not 2.
var minorDigits = map[string]int{
	"JPY": 0,
	"KRW": 0,
	"VND": 0,
	"BHD": 3,
	"KWD": 3,
	"OMR": 3,
	"TND": 3,
}

// Exponent is how many decimal digits the currency's minor unit has (RUB, RSD: 2).
func Exponent(currency string) int {
	if digits, ok := minorDigits[strings.ToUpper(currency)]; ok {
		return digits
	}
	return 2
}

// ParseDecimal reads "1247.5", "-3", "1247,50" into minor units with exp fraction
// digits. Digits beyond exp are rounded half away from zero, in integer arithmetic.
func ParseDecimal(raw string, exp int) (int64, error) {
	s := strings.TrimSpace(raw)
	negative := false
	switch {
	case strings.HasPrefix(s, "-"):
		negative = true
		s = s[1:]
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}
	s = strings.Replace(s, ",", ".", 1)
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" || !digitsOnly(whole) || !digitsOnly(frac) {
		return 0, ErrInvalid
	}
	if whole == "" {
		whole = "0"
	}
	roundUp := false
	if len(frac) > exp {
		roundUp = frac[exp] >= '5'
		frac = frac[:exp]
	}
	frac += strings.Repeat("0", exp-len(frac))
	value, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}
	if roundUp {
		value++
	}
	if negative {
		value = -value
	}
	return value, nil
}

// Rescale moves an integer amount with fromExp fraction digits to toExp digits,
// rounding half away from zero when digits are dropped.
func Rescale(value int64, fromExp, toExp int) int64 {
	for ; fromExp < toExp; fromExp++ {
		value *= 10
	}
	if fromExp == toExp {
		return value
	}
	divisor := int64(1)
	for ; fromExp > toExp; fromExp-- {
		divisor *= 10
	}
	half := divisor / 2
	if value < 0 {
		return -((-value + half) / divisor)
	}
	return (value + half) / divisor
}

func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

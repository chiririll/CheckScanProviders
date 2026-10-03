package money

import "testing"

func TestParseDecimal(t *testing.T) {
	cases := []struct {
		raw  string
		exp  int
		want int64
	}{
		{"1247", 2, 124700},
		{"1247.5", 2, 124750},
		{"1247,50", 2, 124750},
		{"0.1", 2, 10},
		{".05", 2, 5},
		{"0.005", 2, 1},
		{"0.004", 2, 0},
		{"-12.345", 2, -1235},
		{" 99 ", 0, 99},
		{"12345.00", 0, 12345},
	}
	for _, c := range cases {
		got, err := ParseDecimal(c.raw, c.exp)
		if err != nil || got != c.want {
			t.Errorf("ParseDecimal(%q, %d) = %d, %v; want %d", c.raw, c.exp, got, err, c.want)
		}
	}
}

func TestParseDecimalRejectsJunk(t *testing.T) {
	for _, raw := range []string{"", ".", "1.2.3", "1e3", "abc", "1 000"} {
		if _, err := ParseDecimal(raw, 2); err == nil {
			t.Errorf("ParseDecimal(%q) accepted junk", raw)
		}
	}
}

func TestRescale(t *testing.T) {
	if got := Rescale(12345678, 4, 2); got != 123457 {
		t.Errorf("4→2 = %d", got)
	}
	if got := Rescale(-150, 2, 0); got != -2 {
		t.Errorf("negative round = %d", got)
	}
	if got := Rescale(12, 0, 2); got != 1200 {
		t.Errorf("0→2 = %d", got)
	}
}

func TestExponent(t *testing.T) {
	if Exponent("RUB") != 2 || Exponent("rsd") != 2 || Exponent("JPY") != 0 {
		t.Fatal("unexpected exponent")
	}
}

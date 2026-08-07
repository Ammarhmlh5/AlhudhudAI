package excelimport

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		raw      string
		want     string
		wantOK   bool
	}{
		{"0551234567", "+966551234567", true},
		{"966551234567", "+966551234567", true},
		{"+966551234567", "+966551234567", true},
		{"00966551234567", "+966551234567", true},
		{"512345678", "+966512345678", true},
		{" 0551-234 567 ", "+966551234567", true},
		{"(055)1234567", "+966551234567", true},
		{"+971501234567", "+971501234567", true},
		{"770353256", "+967770353256", true},
		{"777893005", "+967777893005", true},
		{"781135727", "+967781135727", true},
		{"912345678", "+967912345678", true},
		{"", "", false},
		{"123", "", false},
		{"abc", "", false},
		{"0551234567890123456", "", false},
	}
	for _, c := range cases {
		got, ok := Normalize(c.raw)
		if ok != c.wantOK || got != c.want {
			t.Errorf("Normalize(%q) = (%q, %v), want (%q, %v)", c.raw, got, ok, c.want, c.wantOK)
		}
	}
}

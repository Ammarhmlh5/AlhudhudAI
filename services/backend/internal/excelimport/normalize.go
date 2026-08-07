package excelimport

import "strings"

// Normalize converts a raw phone string into international format. Local
// mobile numbers without a country code are interpreted with the Saudi (966)
// or Yemeni (967) default, which covers the formats used by this deployment's
// data files. It returns ok=false when the value cannot be interpreted as a
// phone number.
//
// Rules (in priority order):
//   - "+<9-15 digits>"           → kept as-is
//   - "00<country><number>"      → "+" + rest (international prefix)
//   - "0XXXXXXXXX" (10 digits)   → "+966" + rest (Saudi local mobile)
//   - "966XXXXXXXXXX" (12 d)     → "+" + value
//   - "5XXXXXXXX" (9 digits)     → "+966" + value (Saudi mobile w/o leading 0)
//   - "7XXXXXXXX"/"9XXXXXXXX"    → "+967" + value (Yemeni mobile)
func Normalize(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}

	var b strings.Builder
	hasPlus := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && b.Len() == 0:
			hasPlus = true
		}
	}
	d := b.String()
	if d == "" {
		return "", false
	}

	switch {
	case hasPlus && len(d) >= 9 && len(d) <= 15:
		return "+" + d, true
	case strings.HasPrefix(d, "00") && len(d)-2 >= 9 && len(d)-2 <= 15:
		return "+" + d[2:], true
	case strings.HasPrefix(d, "0") && len(d) == 10:
		return "+966" + d[1:], true
	case strings.HasPrefix(d, "966") && len(d) == 12:
		return "+" + d, true
	case len(d) == 9 && strings.HasPrefix(d, "5"):
		return "+966" + d, true
	case len(d) == 9 && (strings.HasPrefix(d, "7") || strings.HasPrefix(d, "9")):
		return "+967" + d, true
	default:
		return "", false
	}
}

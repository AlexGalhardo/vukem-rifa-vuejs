package payments

import "strings"

// MaskEmail keeps the first letter of the mailbox and the domain:
// "maria@example.com" → "m•••@example.com".
func MaskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return "•••"
	}

	return string([]rune(local)[:1]) + "•••@" + domain
}

// MaskPhone keeps the country and area codes and the last four digits:
// "+5511987654321" → "+55 11 •••••-4321".
func MaskPhone(phone string) string {
	d := strings.TrimPrefix(phone, "+")
	if len(d) < 8 {
		return "•••"
	}

	if strings.HasPrefix(d, "55") && len(d) >= 12 {
		return "+55 " + d[2:4] + " •••••-" + d[len(d)-4:]
	}

	return "•••" + d[len(d)-4:]
}

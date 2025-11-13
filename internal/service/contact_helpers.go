package service

import "strings"

func maskEmailAddress(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 2 {
		local = local[:1] + "***"
	} else {
		local = local[:1] + "***" + local[len(local)-1:]
	}
	return local + "@" + domain
}

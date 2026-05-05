package cli

import (
	"os"
	"strings"
)

// MaskEmailEnv toggles email masking across CLI/TUI output (for screenshots).
const MaskEmailEnv = "CCM_MASK_EMAIL"

// MaskEmail returns a redacted version of an email when CCM_MASK_EMAIL=1.
func MaskEmail(email string) string {
	if os.Getenv(MaskEmailEnv) != "1" || email == "" {
		return email
	}
	at := strings.IndexByte(email, '@')
	if at < 1 {
		return "***"
	}
	user := email[:at]
	first := string(user[0])
	return first + "***@***" + topLevel(email[at:])
}

func topLevel(domain string) string {
	dot := strings.LastIndexByte(domain, '.')
	if dot < 0 || dot == len(domain)-1 {
		return ""
	}
	return domain[dot:]
}

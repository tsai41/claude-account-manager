package cli

import "github.com/tsai41/claude-account-manager/internal/format"

// MaskEmail is a thin re-export so existing call sites in this package keep
// working; the implementation lives in internal/format.
func MaskEmail(email string) string { return format.MaskEmail(email) }

package cli

func displayOrDash(s string) string {
	if s == "" || s == "unknown" {
		return "--"
	}
	return s
}

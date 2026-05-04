package usage

import (
	"fmt"
	"strconv"
)

func fmtSscan(s string, v *float64) (int, error) {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	*v = n
	return 1, nil
}

func formatInt(n int) string         { return strconv.Itoa(n) }
func formatFloat(f float64) string   { return fmt.Sprintf("%.1f", f) }

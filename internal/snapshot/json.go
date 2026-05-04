package snapshot

import (
	"encoding/json"
	"io"
)

func jsonEncoder(w io.Writer) *json.Encoder {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e
}

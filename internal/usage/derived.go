package usage

import (
	"github.com/tsai41/claude-account-manager/internal/jsonlscan"
)

// EnrichWithLocalDerived populates ActivityToday, Activity7d, Activity5h, LastActive
// from local jsonl transcripts. Caller decides whether to persist.
func EnrichWithLocalDerived(r *Record) error {
	a, err := jsonlscan.Scan()
	if err != nil {
		return err
	}
	r.ActivityToday = a.Today
	r.Activity7d = a.Last7Days
	r.Activity5h = a.Last5Hours
	r.LastActive = a.LastActive
	return nil
}

// LoadAndDerive returns a Record. If provider == "local-derived", local jsonl scan
// is run and the activity fields are filled in (not persisted automatically).
func LoadAndDerive(profile string) (Record, error) {
	r, err := Load(profile)
	if err != nil {
		return r, err
	}
	if r.Provider == "local-derived" {
		_ = EnrichWithLocalDerived(&r)
	}
	return r, nil
}

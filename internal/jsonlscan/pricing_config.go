package jsonlscan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsai41/claude-account-manager/internal/paths"
)

// PricingFile is the location of the optional user pricing override.
func PricingFile() string { return filepath.Join(paths.CCMRoot(), "pricing.json") }

// PricingEntry is the on-disk representation of one model family override.
type PricingEntry struct {
	Match   string  `json:"match"`              // substring matched against the model id (case-insensitive)
	Family  string  `json:"family"`             // display name
	Pricing Pricing `json:"pricing"`            // override numbers
}

// LoadPricingOverrides reads ~/.ccm/pricing.json (if present) and returns the user
// overrides. Format:
//
//	[
//	  {"match":"opus","family":"Opus","pricing":{"input_per_m":15,"output_per_m":75,
//	    "cache_create_5m_mult":1.25,"cache_create_1h_mult":2.0,"cache_read_mult":0.1}},
//	  ...
//	]
//
// Empty/missing file returns nil overrides; defaults still apply.
func LoadPricingOverrides() ([]PricingEntry, error) {
	b, err := os.ReadFile(PricingFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []PricingEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// resolvedPricing returns (family, pricing, ok) for a model id, applying any
// loaded pricing overrides first and falling back to DefaultPricing.
func resolvedPricing(model string, overrides []PricingEntry) (string, Pricing, bool) {
	low := strings.ToLower(model)
	for _, e := range overrides {
		if e.Match == "" {
			continue
		}
		if strings.Contains(low, strings.ToLower(e.Match)) {
			fam := e.Family
			if fam == "" {
				fam = e.Match
			}
			return fam, e.Pricing, true
		}
	}
	return familyOf(model)
}

// WriteDefaultPricingFile writes a starter pricing.json the user can edit. It does
// NOT overwrite an existing file.
func WriteDefaultPricingFile() (string, error) {
	if _, err := os.Stat(PricingFile()); err == nil {
		return PricingFile(), nil
	}
	if err := os.MkdirAll(filepath.Dir(PricingFile()), 0o700); err != nil {
		return "", err
	}
	entries := make([]PricingEntry, 0, len(DefaultPricing))
	for _, e := range DefaultPricing {
		entries = append(entries, PricingEntry{Match: e.Match, Family: e.Family, Pricing: e.Pricing})
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(PricingFile(), b, 0o600); err != nil {
		return "", err
	}
	return PricingFile(), nil
}

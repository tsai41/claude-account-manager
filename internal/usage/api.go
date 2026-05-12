package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// OAuthUsageURL is the Anthropic OAuth usage endpoint mirrored from CCSwitcher.
const OAuthUsageURL = "https://api.anthropic.com/api/oauth/usage"

// OAuthBeta is the anthropic-beta header value required by the OAuth usage endpoint.
const OAuthBeta = "oauth-2025-04-20"

// ErrTokenExpired is returned when the access token is rejected by the API.
var ErrTokenExpired = errors.New("oauth access token expired")

// ErrRateLimited is returned when the API returns 429.
var ErrRateLimited = errors.New("oauth usage rate limited")

type oauthWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type oauthUsageResp struct {
	FiveHour *oauthWindow `json:"five_hour"`
	SevenDay *oauthWindow `json:"seven_day"`
}

// OAuthUsage is the decoded result of an OAuth usage call.
type OAuthUsage struct {
	SessionUtilization float64
	WeeklyUtilization  float64
	SessionResetsAt    time.Time
	WeeklyResetsAt     time.Time
}

// FetchOAuthUsage calls the Anthropic OAuth usage endpoint with the given access token.
// It retries once on HTTP 429, honoring Retry-After when present (capped at 30s).
func FetchOAuthUsage(ctx context.Context, accessToken string) (OAuthUsage, error) {
	var out OAuthUsage
	if accessToken == "" {
		return out, errors.New("empty access token")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	body, status, retryAfter, err := doOAuthUsageRequest(ctx, accessToken)
	if err != nil {
		return out, err
	}
	if status == http.StatusTooManyRequests {
		wait := retryAfter
		if wait <= 0 {
			wait = 5 * time.Second
		}
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return out, ctx.Err()
		}
		body, status, _, err = doOAuthUsageRequest(ctx, accessToken)
		if err != nil {
			return out, err
		}
		if status == http.StatusTooManyRequests {
			return out, ErrRateLimited
		}
	}
	if status == http.StatusUnauthorized {
		return out, ErrTokenExpired
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("oauth usage HTTP %d", status)
	}
	var raw oauthUsageResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return out, fmt.Errorf("oauth usage decode: %w", err)
	}
	if raw.FiveHour != nil {
		out.SessionUtilization = raw.FiveHour.Utilization
		out.SessionResetsAt = parseResetTime(raw.FiveHour.ResetsAt)
	}
	if raw.SevenDay != nil {
		out.WeeklyUtilization = raw.SevenDay.Utilization
		out.WeeklyResetsAt = parseResetTime(raw.SevenDay.ResetsAt)
	}
	return out, nil
}

func parseResetTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ApplyOAuth merges fetched OAuth data into the record and persists it.
func ApplyOAuth(profile string, u OAuthUsage) error {
	r, err := Load(profile)
	if err != nil {
		return err
	}
	r.Provider = "oauth"
	r.Session = Field{Display: pctDisplay(u.SessionUtilization), Source: "oauth"}
	r.Weekly = Field{Display: pctDisplay(u.WeeklyUtilization), Source: "oauth"}
	r.SessionResetsAt = u.SessionResetsAt
	r.WeeklyResetsAt = u.WeeklyResetsAt
	return Save(profile, r)
}

func pctDisplay(v float64) string {
	if v <= 0 {
		return "0%"
	}
	if v == float64(int(v)) {
		return strconv.Itoa(int(v)) + "%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

// FormatReset renders a single resets_at deadline as a short countdown like "3h12m".
// Returns "--" when zero, "due" when past.
func FormatReset(t time.Time) string {
	if t.IsZero() {
		return "--"
	}
	d := time.Until(t)
	if d <= 0 {
		return "due"
	}
	if d >= 24*time.Hour {
		days := int(d / (24 * time.Hour))
		hours := int((d % (24 * time.Hour)) / time.Hour)
		if hours == 0 {
			return strconv.Itoa(days) + "d"
		}
		return strconv.Itoa(days) + "d" + strconv.Itoa(hours) + "h"
	}
	if d >= time.Hour {
		h := int(d / time.Hour)
		m := int((d % time.Hour) / time.Minute)
		if m == 0 {
			return strconv.Itoa(h) + "h"
		}
		return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m"
	}
	m := int(d / time.Minute)
	if m <= 0 {
		return "<1m"
	}
	return strconv.Itoa(m) + "m"
}

// doOAuthUsageRequest performs a single HTTP call. Returns body bytes, status,
// parsed Retry-After (if any), and a transport error if the request itself failed.
func doOAuthUsageRequest(ctx context.Context, accessToken string) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, OAuthUsageURL, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", OAuthBeta)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("oauth usage: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var retryAfter time.Duration
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			retryAfter = time.Duration(secs) * time.Second
		}
	}
	return body, resp.StatusCode, retryAfter, nil
}

// FormatResetPair renders both session + weekly resets as "S:3h W:5d2h".
func FormatResetPair(session, weekly time.Time) string {
	if session.IsZero() && weekly.IsZero() {
		return "--"
	}
	return "S:" + FormatReset(session) + " W:" + FormatReset(weekly)
}

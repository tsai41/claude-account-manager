package usage

import "testing"

func TestParseManual(t *testing.T) {
	cases := []struct {
		in           string
		wantSession  string
		wantWeekly   string
	}{
		{"session 42%, weekly 68%", "42%", "68%"},
		{"WEEKLY 5%, session 10%", "10%", "5%"},
		{"session 0.5%, weekly 99%", "0.5%", "99%"},
		{"plain string", "", ""},
		{"weekly only 30%", "", "30%"},
		{"session only 7%", "7%", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			s, w := ParseManual(c.in)
			if s != c.wantSession || w != c.wantWeekly {
				t.Fatalf("ParseManual(%q) = (%q, %q), want (%q, %q)", c.in, s, w, c.wantSession, c.wantWeekly)
			}
		})
	}
}

func TestRemaining(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"42%", "58%"},
		{"0%", "100%"},
		{"100%", "0%"},
		{"99.5%", "0.5%"},
		{"unknown", "--"},
		{"", "--"},
		{"--", "--"},
		{"abc", "abc"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := Remaining(c.in)
			if got != c.want {
				t.Fatalf("Remaining(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

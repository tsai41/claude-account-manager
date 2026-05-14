package dirmap

import (
	"path/filepath"
	"testing"
)

func TestResolveLongestPrefixWins(t *testing.T) {
	m := Map{Bindings: []Binding{
		{Profile: "max", Pattern: "/home/u/work"},
		{Profile: "team", Pattern: "/home/u/work/gobe"},
		{Profile: "personal", Pattern: "/home/u/personal"},
	}}
	cases := []struct {
		dir  string
		want string
	}{
		{"/home/u/work", "max"},
		{"/home/u/work/foo", "max"},
		{"/home/u/work/gobe", "team"},
		{"/home/u/work/gobe/sub/deeper", "team"},
		{"/home/u/personal/stuff", "personal"},
		{"/home/u/elsewhere", ""},
		{"/", ""},
	}
	for _, c := range cases {
		if got := m.Resolve(c.dir); got != c.want {
			t.Errorf("Resolve(%q) = %q, want %q", c.dir, got, c.want)
		}
	}
}

func TestPatternMatchesBoundary(t *testing.T) {
	// /home/u/work must NOT match /home/u/workspace (no path-component bleed).
	m := Map{Bindings: []Binding{{Profile: "p", Pattern: "/home/u/work"}}}
	if got := m.Resolve("/home/u/workspace"); got != "" {
		t.Errorf("Resolve(/home/u/workspace) = %q, want empty (no prefix bleed)", got)
	}
	if got := m.Resolve("/home/u/work"); got != "p" {
		t.Errorf("Resolve(/home/u/work) = %q, want %q", got, "p")
	}
}

func TestBindReplacesExistingPattern(t *testing.T) {
	var m Map
	if err := m.Bind("a", "/x"); err != nil {
		t.Fatal(err)
	}
	if err := m.Bind("b", "/x"); err != nil {
		t.Fatal(err)
	}
	if len(m.Bindings) != 1 {
		t.Fatalf("expected 1 binding after replace, got %d", len(m.Bindings))
	}
	if m.Bindings[0].Profile != "b" {
		t.Errorf("expected profile=b after replace, got %q", m.Bindings[0].Profile)
	}
}

func TestUnbind(t *testing.T) {
	var m Map
	_ = m.Bind("p", "/x")
	if !m.Unbind("/x") {
		t.Errorf("Unbind(/x) returned false, want true")
	}
	if len(m.Bindings) != 0 {
		t.Errorf("expected empty after unbind, got %d entries", len(m.Bindings))
	}
	if m.Unbind("/missing") {
		t.Errorf("Unbind(/missing) returned true, want false")
	}
}

func TestCanonicalizeTilde(t *testing.T) {
	// ~/foo should expand under HOME; the result must be absolute and cleaned.
	c := canonicalize("~/foo/../bar")
	if !filepath.IsAbs(c) {
		t.Errorf("canonicalize(~/foo/../bar) not absolute: %q", c)
	}
	if filepath.Base(c) != "bar" {
		t.Errorf("canonicalize did not clean .. — got %q", c)
	}
}

package profile

import "testing"

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"work", false},
		{"work-1", false},
		{"work_2", false},
		{"WORK", false},
		{"123", false},
		{"", true},
		{".", true},
		{"..", true},
		{"with space", true},
		{"with/slash", true},
		{"with.dot", true},
		{"hello!", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateName(c.name)
			if c.wantErr && err == nil {
				t.Fatalf("ValidateName(%q) err = nil, want error", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("ValidateName(%q) err = %v, want nil", c.name, err)
			}
		})
	}
}

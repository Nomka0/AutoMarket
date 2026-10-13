package domain_test

import (
	"testing"

	"automarket/internal/domain"
)

func TestIsValidStatus(t *testing.T) {
	cases := []struct {
		status string
		valid  bool
	}{
		{domain.StatusPending, true},
		{domain.StatusApproved, true},
		{domain.StatusSold, true},
		{"REJECTED", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := domain.IsValidStatus(tc.status); got != tc.valid {
			t.Errorf("IsValidStatus(%q) = %v; want %v", tc.status, got, tc.valid)
		}
	}
}

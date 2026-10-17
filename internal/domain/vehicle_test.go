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

func TestContactHelpers(t *testing.T) {
	email, phone := domain.ParseContact("juan@gmail.com / 3001234567")
	if email != "juan@gmail.com" || phone != "3001234567" {
		t.Errorf("unexpected parsed contact: email=%q, phone=%q", email, phone)
	}

	email2, phone2 := domain.ParseContact("3001234567 / juan@gmail.com")
	if email2 != "juan@gmail.com" || phone2 != "3001234567" {
		t.Errorf("unexpected parsed contact inverted: email=%q, phone=%q", email2, phone2)
	}

	email3, phone3 := domain.ParseContact("juan@gmail.com")
	if email3 != "juan@gmail.com" || phone3 != "" {
		t.Errorf("unexpected email only: email=%q, phone=%q", email3, phone3)
	}

	formatted := domain.FormatContact("juan@gmail.com", "3001234567")
	if formatted != "juan@gmail.com / 3001234567" {
		t.Errorf("unexpected formatted contact: %q", formatted)
	}

	pub := domain.VehiclePublication{
		SellerContact: "juan@gmail.com / +57 300 123-4567",
	}
	if pub.ContactEmail() != "juan@gmail.com" {
		t.Errorf("expected juan@gmail.com, got %q", pub.ContactEmail())
	}
	if pub.ContactPhone() != "+57 300 123-4567" {
		t.Errorf("expected +57 300 123-4567, got %q", pub.ContactPhone())
	}
	if pub.CleanPhone() != "+573001234567" {
		t.Errorf("expected +573001234567, got %q", pub.CleanPhone())
	}
	if pub.WhatsAppPhone() != "573001234567" {
		t.Errorf("expected 573001234567, got %q", pub.WhatsAppPhone())
	}

	pubWithoutPrefix := domain.VehiclePublication{
		Title:         "Mazda 3 2020",
		SellerContact: "ana@gmail.com / 3109876543",
	}
	if pubWithoutPrefix.CleanPhone() != "+573109876543" {
		t.Errorf("expected +573109876543, got %q", pubWithoutPrefix.CleanPhone())
	}
	if pubWithoutPrefix.WhatsAppPhone() != "573109876543" {
		t.Errorf("expected 573109876543, got %q", pubWithoutPrefix.WhatsAppPhone())
	}
	if pubWithoutPrefix.DefaultMessage() != "Hola, me interesa el Mazda 3 2020 que estás vendiendo" {
		t.Errorf("unexpected default message: %q", pubWithoutPrefix.DefaultMessage())
	}
	if pubWithoutPrefix.MailtoURL() == "" || pubWithoutPrefix.WhatsAppURL() == "" {
		t.Errorf("expected non-empty mailto and whatsapp URLs")
	}
}

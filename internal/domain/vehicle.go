package domain

import (
	"net/url"
	"strings"
	"time"
)

const (
	StatusPending  = "PENDING"
	StatusApproved = "APPROVED"
	StatusSold     = "SOLD"
)

type VehiclePublication struct {
	ID            string    `json:"id"`
	SellerID      string    `json:"seller_id"`
	SellerContact string    `json:"seller_contact"`
	SellerEmail   string    `json:"seller_email,omitempty"`
	SellerPhone   string    `json:"seller_phone,omitempty"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	Year          int       `json:"year"`
	ImageURL      string    `json:"image_url,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func IsValidStatus(status string) bool {
	return status == StatusPending || status == StatusApproved || status == StatusSold
}

func (v VehiclePublication) ContactEmail() string {
	if v.SellerEmail != "" {
		return v.SellerEmail
	}
	email, _ := ParseContact(v.SellerContact)
	return email
}

func (v VehiclePublication) ContactPhone() string {
	if v.SellerPhone != "" {
		return v.SellerPhone
	}
	_, phone := ParseContact(v.SellerContact)
	return phone
}

func (v VehiclePublication) CleanPhone() string {
	phone := v.ContactPhone()
	var b strings.Builder
	for _, r := range phone {
		if (r >= '0' && r <= '9') || r == '+' {
			b.WriteRune(r)
		}
	}
	res := b.String()
	if strings.HasPrefix(res, "+") {
		return res
	}
	if len(res) == 10 {
		return "+57" + res
	}
	if len(res) > 0 {
		return "+57" + res
	}
	return res
}

func (v VehiclePublication) WhatsAppPhone() string {
	return strings.TrimPrefix(v.CleanPhone(), "+")
}

func (v VehiclePublication) DefaultMessage() string {
	return "Hola, me interesa el " + v.Title + " que estás vendiendo"
}

func (v VehiclePublication) MailtoURL() string {
	email := v.ContactEmail()
	if email == "" {
		return ""
	}
	subject := "Interés en " + v.Title
	body := v.DefaultMessage()
	return "mailto:" + email + "?subject=" + url.QueryEscape(subject) + "&body=" + url.QueryEscape(body)
}

func (v VehiclePublication) WhatsAppURL() string {
	phone := v.WhatsAppPhone()
	if phone == "" {
		return ""
	}
	return "https://wa.me/" + phone + "?text=" + url.QueryEscape(v.DefaultMessage())
}


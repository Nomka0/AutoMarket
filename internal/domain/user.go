package domain

import "strings"

const (
	RoleSeller = "vendedor"
	RoleAdmin  = "admin"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Salt         string `json:"salt"`
	Role         string `json:"role"`
	ContactInfo  string `json:"contact_info"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

func (u User) GetEmail() string {
	if u.Email != "" {
		return u.Email
	}
	email, _ := ParseContact(u.ContactInfo)
	return email
}

func (u User) GetPhone() string {
	if u.Phone != "" {
		return u.Phone
	}
	_, phone := ParseContact(u.ContactInfo)
	return phone
}

func FormatContact(email, phone string) string {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if email != "" && phone != "" {
		return email + " / " + phone
	}
	if email != "" {
		return email
	}
	return phone
}

func ParseContact(contact string) (email, phone string) {
	contact = strings.TrimSpace(contact)
	if contact == "" || contact == "N/A" || contact == "No especificado" {
		return "", ""
	}
	for _, sep := range []string{" / ", " | ", "\n", ", "} {
		if strings.Contains(contact, sep) {
			parts := strings.SplitN(contact, sep, 2)
			p1, p2 := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if strings.Contains(p1, "@") {
				return p1, p2
			} else if strings.Contains(p2, "@") {
				return p2, p1
			}
			return p1, p2
		}
	}
	if strings.Contains(contact, "@") {
		return contact, ""
	}
	return "", contact
}

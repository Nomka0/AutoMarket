package domain

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
}

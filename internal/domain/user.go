package domain

const (
	RoleSeller = "vendedor"
	RoleAdmin  = "admin"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Salt         string `json:"-"`
	Role         string `json:"role"`
	ContactInfo  string `json:"contact_info"`
}

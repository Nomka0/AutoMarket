package domain

import "time"

const (
	StatusPending  = "PENDING"
	StatusApproved = "APPROVED"
	StatusSold     = "SOLD"
)

type VehiclePublication struct {
	ID            string    `json:"id"`
	SellerID      string    `json:"seller_id"`
	SellerContact string    `json:"seller_contact"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	Year          int       `json:"year"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func IsValidStatus(status string) bool {
	return status == StatusPending || status == StatusApproved || status == StatusSold
}

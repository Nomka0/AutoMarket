package handlers

type CreatePublicationRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Year        int     `json:"year" binding:"required,gt=1900"`
}

package dto

type UserResponse struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Confirmed bool   `json:"confirmed"`
	Deleted   bool   `json:"deleted"`
}

type DeleteUserResponse struct {
	Anonymized bool `json:"anonymized"`
}

package dto

type MeResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ConfirmRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

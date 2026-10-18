package dto

type PayRequest struct {
	CardNumber string `json:"card_number"`
	Expiry     string `json:"expiry"`
	CVC        string `json:"cvc"`
}

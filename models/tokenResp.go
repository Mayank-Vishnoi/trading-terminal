package models

type TokenResponse struct {
	Email         string   `json:"email"`
	Exchanges     []string `json:"exchanges"`
	Products      []string `json:"products"`
	Broker        string   `json:"broker"`
	UserID        string   `json:"user_id"`
	UserName      string   `json:"user_name"`
	OrderTypes    []string `json:"order_types"`
	UserType      string   `json:"user_type"`
	POA           bool     `json:"poa"`
	IsActive      bool     `json:"is_active"`
	AccessToken   string   `json:"access_token"`
	ExtendedToken string   `json:"extended_token"`
}
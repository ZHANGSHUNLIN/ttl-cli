package models

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	APIKey    string `json:"api_key"`
	CreatedAt int64  `json:"created_at"`
	Active    bool   `json:"active"`
}

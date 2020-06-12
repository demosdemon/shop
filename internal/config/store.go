package config

type Store struct {
	File     string `json:"file"`
	ID       string `json:"id"`
	StoreID  string `json:"store_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

package dto

// Locale represents a locale file organized by domain and translation key.
type Locale struct {
	Code    string                       `json:"code"`
	Domains map[string]map[string]string `json:"domains"`
}

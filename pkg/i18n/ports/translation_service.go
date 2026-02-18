package ports

// TranslationService provides translation lookups for configured locale data.
type TranslationService interface {
	Translate(domain, key string) string
	TranslateWithData(domain, key string, data map[string]string) string
	GetAllForDomain(domain string) map[string]string
}

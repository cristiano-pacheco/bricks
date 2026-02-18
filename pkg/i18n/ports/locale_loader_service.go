package ports

// LocaleLoaderService loads locale data for a given locale code.
type LocaleLoaderService interface {
	Load(locale string) (map[string]map[string]string, error)
}

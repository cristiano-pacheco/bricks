package itestkit

// Config holds configuration for test containers.
type Config struct {
	PostgresImage  string
	RedisImage     string
	MigrationsPath string
	Database       string
	User           string
	Password       string //nolint:gosec // intentional config field for test container credentials
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PostgresImage:  "postgres:16-alpine",
		RedisImage:     "redis:7-alpine",
		MigrationsPath: "file://migrations",
		Database:       "itest",
		User:           "itest",
		Password:       "itest",
	}
}

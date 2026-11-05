package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/database"
	"github.com/cristiano-pacheco/bricks/pkg/migration"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var embedded embed.FS

// schemaReady orders hook registration, not database work in constructors.
type schemaReady struct{}

type migrationInputs struct {
	fx.In
	Files []migration.FileSystem `group:"application_migrations"`
}

func migrationFiles() (migration.FileSystem, error) {
	root, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return migration.FileSystem{}, err
	}
	return migration.New(root), nil
}

func provideRunner(in migrationInputs) *migration.Runner {
	return migration.NewRunner(in.Files)
}

// migrationURL uses URL escaping rather than Config.PostgresDSN's interpolation.
func migrationURL(cfg database.Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("sslmode", "disable")
	if cfg.SSLMode {
		query.Set("sslmode", "require")
		for key, value := range map[string]string{
			"sslcert": cfg.SSLCert, "sslkey": cfg.SSLKey, "sslrootcert": cfg.SSLRootCert,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
	}
	timeZone := cfg.TimeZone
	if strings.TrimSpace(timeZone) == "" {
		timeZone = "UTC"
	}
	query.Set("TimeZone", timeZone)
	for key, value := range map[string]string{
		"application_name": cfg.ApplicationName, "search_path": cfg.SearchPath,
	} {
		if value != "" {
			query.Set(key, value)
		}
	}
	for key, value := range map[string]int{
		"connect_timeout":                     cfg.ConnectTimeout,
		"statement_timeout":                   cfg.StatementTimeout,
		"lock_timeout":                        cfg.LockTimeout,
		"idle_in_transaction_session_timeout": cfg.IdleInTransaction,
	} {
		if value > 0 {
			query.Set(key, strconv.Itoa(value))
		}
	}
	target := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, strconv.FormatUint(uint64(cfg.Port), 10)),
		Path:     "/" + cfg.Name,
		RawQuery: query.Encode(),
	}
	return target.String(), nil
}

func registerMigrations(
	lifecycle fx.Lifecycle,
	runner *migration.Runner,
	cfg config.Config[database.Config],
	_ *gorm.DB,
) (*schemaReady, error) {
	dsn, err := migrationURL(cfg.Get())
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if err := runner.Up(dsn); err != nil {
				return fmt.Errorf("apply PostgreSQL migrations: %w", err)
			}
			return nil
		},
	})
	return &schemaReady{}, nil
}

func registerApplication(lifecycle fx.Lifecycle, db *gorm.DB, _ *schemaReady) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var count int64
			if err := db.WithContext(ctx).Table("bricks_skill_startup_items").Count(&count).Error; err != nil {
				return fmt.Errorf("query migrated table: %w", err)
			}
			return nil
		},
	})
}

func run() error {
	app := fx.New(
		database.Module,
		fx.Provide(
			fx.Annotate(migrationFiles, fx.ResultTags(`group:"application_migrations"`)),
			provideRunner,
			registerMigrations,
		),
		fx.Invoke(registerApplication),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStop()
	return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

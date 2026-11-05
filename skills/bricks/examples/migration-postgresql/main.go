package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/migration"
)

//go:embed migrations/*.sql
var embedded embed.FS

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

func registerMigration(lifecycle fx.Lifecycle, runner *migration.Runner) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			dsn := os.Getenv("MIGRATION_DSN")
			if dsn == "" {
				return errors.New("MIGRATION_DSN is required")
			}

			return runner.Up(dsn)
		},
	})
}

func run() (err error) {
	app := fx.New(
		fx.Provide(
			fx.Annotate(migrationFiles, fx.ResultTags(`group:"application_migrations"`)),
			provideRunner,
		),
		fx.Invoke(registerMigration),
	)
	if err := app.Err(); err != nil {
		return fmt.Errorf("construct migration application: %w", err)
	}
	defer func() {
		if stopErr := app.Stop(context.Background()); stopErr != nil {
			err = errors.Join(err, fmt.Errorf("stop migration application: %w", stopErr))
		}
	}()

	if err := app.Start(context.Background()); err != nil {
		return fmt.Errorf("start migration application: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

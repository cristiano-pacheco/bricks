package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"time"

	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
	"github.com/cristiano-pacheco/bricks/pkg/migration"
)

//go:embed migrations/*.sql
var embedded embed.FS

type schemaReady struct{}

func provideRunner() (*migration.Runner, error) {
	root, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return nil, err
	}

	return migration.NewRunner([]migration.FileSystem{migration.New(root)}), nil
}

func run(down bool) error {
	registerMigrations := func(lifecycle fx.Lifecycle, runner *migration.Runner, db *gorm.DB) (*schemaReady, error) {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		lifecycle.Append(fx.Hook{
			OnStart: func(context.Context) error {
				if down {
					return runner.DownSQLite(sqlDB)
				}

				return runner.UpSQLite(sqlDB)
			},
		})

		return &schemaReady{}, nil
	}
	registerApplication := func(lifecycle fx.Lifecycle, db *gorm.DB, _ *schemaReady) {
		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if down {
					fmt.Println("development rollback complete")
					return nil
				}

				var count int64
				if err := db.WithContext(ctx).Table("bricks_skill_items").Count(&count).Error; err != nil {
					return fmt.Errorf("query migrated table: %w", err)
				}

				fmt.Printf("items=%d\n", count)

				return nil
			},
		})
	}

	app := fx.New(
		sqlite.Module,
		fx.Provide(provideRunner, registerMigrations),
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
	down := flag.Bool("down", false, "undo all migrations on a disposable development database")

	flag.Parse()

	if err := run(*down); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

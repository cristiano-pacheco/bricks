package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/database"
	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
	"gorm.io/gorm"
)

func newPostgres(lc fx.Lifecycle, cfg config.Config[database.Config]) (*gorm.DB, error) {
	return database.NewWithLifecycle(cfg.Get(), lc)
}

type databases struct {
	fx.In
	Postgres *gorm.DB `name:"postgres"`
	SQLite   *gorm.DB `name:"sqlite"`
}

func registerWork(lc fx.Lifecycle, db databases) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		var postgresValue int
		if err := db.Postgres.WithContext(ctx).Raw("SELECT 1").Scan(&postgresValue).Error; err != nil {
			return fmt.Errorf("query PostgreSQL: %w", err)
		}
		var sqliteValue int
		if err := db.SQLite.WithContext(ctx).Raw("SELECT 2").Scan(&sqliteValue).Error; err != nil {
			return fmt.Errorf("query SQLite: %w", err)
		}
		if postgresValue != 1 || sqliteValue != 2 {
			return fmt.Errorf("unexpected results: postgres=%d sqlite=%d", postgresValue, sqliteValue)
		}
		fmt.Printf("postgres=%d sqlite=%d\n", postgresValue, sqliteValue)
		return nil
	}})
}

func run() error {
	app := fx.New(
		config.Provide[database.Config]("app.database"),
		config.Provide[sqlite.Config]("app.sqlite"),
		fx.Provide(fx.Annotate(newPostgres, fx.ResultTags(`name:"postgres"`))),
		fx.Provide(fx.Annotate(sqlite.NewWithLifecycle, fx.ResultTags(`name:"sqlite"`))),
		fx.Invoke(registerWork),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

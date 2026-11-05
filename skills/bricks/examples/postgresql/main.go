package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/database"
)

func registerQuery(lifecycle fx.Lifecycle, db *gorm.DB) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var value int
			if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
				return fmt.Errorf("query PostgreSQL: %w", err)
			}

			if value != 1 {
				return errors.New("unexpected PostgreSQL query result")
			}

			return nil
		},
	})
}

func run() error {
	app := fx.New(database.Module, fx.Invoke(registerQuery))
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

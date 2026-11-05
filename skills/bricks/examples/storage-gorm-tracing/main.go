package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/database"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func query(ctx context.Context, db *gorm.DB) error {
	ctx, span := brickstrace.Span(ctx, "check-postgres")
	defer span.End()
	var value int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
		return fmt.Errorf("query PostgreSQL: %w", err)
	}
	if value != 1 {
		return fmt.Errorf("unexpected query result: %d", value)
	}
	fmt.Println("postgres=1")
	return nil
}

func registerWork(lc fx.Lifecycle, db *gorm.DB) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return query(ctx, db)
	}})
}

func run() error {
	app := fx.New(
		brickstrace.GORMModule(),
		database.Module,
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

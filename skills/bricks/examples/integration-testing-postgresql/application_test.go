//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type greetingService struct{ db *gorm.DB }

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var greeting string
	if err := s.db.WithContext(ctx).Raw("SELECT 'hello ' || ?::text", name).Scan(&greeting).Error; err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	return greeting, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	var service *greetingService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Provide(func(db *gorm.DB) *greetingService { return &greetingService{db: db} }),
		fx.Populate(&service),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = errors.Join(err, app.Stop(ctx))
	}()
	if err := app.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		return err
	}
	greeting, err := service.Execute(ctx, "Ada")
	if err != nil {
		return err
	}
	if greeting != "hello Ada" {
		return fmt.Errorf("unexpected greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestPostgresApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}

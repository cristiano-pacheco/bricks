//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type userService struct{ db *gorm.DB }

func (s *userService) Register(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var saved string
	if err := s.db.WithContext(ctx).Raw("INSERT INTO users (name) VALUES (?) RETURNING name", name).Scan(&saved).Error; err != nil {
		return "", fmt.Errorf("register user: %w", err)
	}
	return saved, nil
}

func exercise() (err error) {
	path, err := filepath.Abs("migrations")
	if err != nil {
		return fmt.Errorf("resolve migration directory: %w", err)
	}
	cfg := itestkit.DefaultConfig()
	cfg.MigrationsPath = (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	kit := itestkit.New(cfg)
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	if err := kit.RunMigrations(); err != nil {
		return err
	}
	var service *userService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Provide(func(db *gorm.DB) *userService { return &userService{db: db} }),
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
	saved, err := service.Register(ctx, "Ada")
	if err != nil {
		return err
	}
	if saved != "Ada" {
		return fmt.Errorf("unexpected saved name %q", saved)
	}
	if _, err := service.Register(ctx, "Ada"); err == nil {
		return errors.New("duplicate registration did not fail")
	}
	return nil
}

func TestMigratedApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}

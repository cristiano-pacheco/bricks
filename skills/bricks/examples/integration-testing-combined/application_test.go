//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	upstreamredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type greetingService struct {
	db *gorm.DB
	cache *bricksredis.Client
}

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var greeting string
	if err := s.db.WithContext(ctx).Raw("SELECT 'hello ' || ?::text", name).Scan(&greeting).Error; err != nil {
		return "", fmt.Errorf("query greeting: %w", err)
	}
	key := s.cache.WithNamespace("greeting")
	raw := s.cache.UniversalClient()
	if err := raw.Set(ctx, key, greeting, time.Minute).Err(); err != nil {
		return "", fmt.Errorf("cache greeting: %w", err)
	}
	value, err := raw.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("read cached greeting: %w", err)
	}
	return value, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	if err := kit.StartRedis(); err != nil {
		return err
	}
	raw, ok := kit.Redis().(*upstreamredis.Client)
	if !ok {
		return fmt.Errorf("expected standalone kit client, got %T", kit.Redis())
	}
	cfg := bricksredis.Config{
		Type: bricksredis.ClientTypeSingleNode,
		URL: (&url.URL{Scheme: "redis", Host: raw.Options().Addr}).String(),
		Namespace: "combined-test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cache, err := bricksredis.NewClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, cache.Close()) }()
	var service *greetingService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Supply(cache),
		fx.Provide(func(db *gorm.DB, cache *bricksredis.Client) *greetingService {
			return &greetingService{db: db, cache: cache}
		}),
		fx.Populate(&service),
	)
	defer func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelStop()
		err = errors.Join(err, app.Stop(stopCtx))
	}()
	if err := app.Err(); err != nil {
		return err
	}
	if err := app.Start(ctx); err != nil {
		return err
	}
	greeting, err := service.Execute(ctx, "Ada")
	if err != nil {
		return err
	}
	if greeting != "hello Ada" {
		return fmt.Errorf("unexpected cached greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestCombinedApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}

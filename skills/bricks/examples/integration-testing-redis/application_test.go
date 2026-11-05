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
)

type greetingService struct{ client *bricksredis.Client }

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	key := s.client.WithNamespace("greeting")
	raw := s.client.UniversalClient()
	if err := raw.Set(ctx, key, "hello "+name, time.Minute).Err(); err != nil {
		return "", fmt.Errorf("store greeting: %w", err)
	}
	value, err := raw.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	return value, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
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
		Namespace: "greeting-test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := bricksredis.NewClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, client.Close()) }()
	var service *greetingService
	app := fx.New(
		fx.Supply(client),
		fx.Provide(func(client *bricksredis.Client) *greetingService { return &greetingService{client: client} }),
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
		return fmt.Errorf("unexpected greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestRedisApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}

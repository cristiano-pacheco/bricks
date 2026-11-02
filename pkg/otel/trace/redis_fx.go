package trace

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// RedisModule instruments an application-owned Redis client when enabled. Include
// Module or GORMModule separately to manage the tracing provider.
func RedisModule() fx.Option {
	cfg, err := config.New[redisYAMLConfig]()
	if err != nil {
		return fx.Error(fmt.Errorf("load Redis tracing configuration: %w", err))
	}

	if !cfg.Get().App.Redis.Tracing.Enabled {
		return fx.Options()
	}

	return fx.Module("otel.trace.redis",
		fx.Provide(newRedisLifecycle, func() *bricksredis.TracingEnabled { return &bricksredis.TracingEnabled{} }),
		fx.Invoke(func(*redisLifecycle) {}),
		fx.Invoke(instrumentRedis),
	)
}

type redisYAMLConfig struct {
	App struct {
		Redis struct {
			Tracing struct {
				Enabled bool `config:"enabled"`
			} `config:"tracing"`
		} `config:"redis"`
	} `config:"app"`
}

type redisInstrumentation struct {
	active  atomic.Bool
	mu      sync.RWMutex
	tracer  oteltrace.Tracer
	nodesMu sync.Mutex
	nodes   map[weak.Pointer[redis.Client]]struct{}
}

// Redis has no hook removal API. Weak keys allow a hook to be reused across
// application restarts without keeping an otherwise unreachable client alive.
//
//nolint:gochecknoglobals // Hook reuse requires a process-wide index; keys do not retain clients.
var redisHooks = struct {
	mu sync.Mutex

	clients map[weak.Pointer[redis.Client]]*redisInstrumentation
}{clients: make(map[weak.Pointer[redis.Client]]*redisInstrumentation)}

type redisLifecycle struct {
	client    *redis.Client
	key       weak.Pointer[redis.Client]
	installed *redisInstrumentation
}

type redisParams struct {
	fx.In

	Instrumentation *redisLifecycle
	Client          redis.UniversalClient
}

func newRedisLifecycle(lc fx.Lifecycle, _ *traceLifecycle) *redisLifecycle {
	i := &redisLifecycle{}
	lc.Append(fx.Hook{OnStart: i.start, OnStop: i.stop})

	return i
}

func instrumentRedis(params redisParams) error {
	var (
		key    weak.Pointer[redis.Client]
		client *redis.Client
	)

	switch c := params.Client.(type) {
	case *redis.Client:
		if c == nil {
			return errors.New("redis tracing requires an initialized client")
		}

		key = weak.Make(c)
		client = c
	case *redis.ClusterClient:
		return errors.New("redis tracing: cluster clients are not supported")
	case *redis.Ring:
		return errors.New("redis tracing: ring clients are not supported")
	default:
		return fmt.Errorf("redis tracing: unsupported client %T", params.Client)
	}

	params.Instrumentation.client = client
	params.Instrumentation.key = key

	return nil
}

func (i *redisLifecycle) start(_ context.Context) error {
	redisHooks.mu.Lock()
	defer redisHooks.mu.Unlock()

	for k := range redisHooks.clients {
		if k.Value() == nil {
			delete(redisHooks.clients, k)
		}
	}

	state := redisHooks.clients[i.key]
	if state == nil {
		state = &redisInstrumentation{nodes: make(map[weak.Pointer[redis.Client]]struct{})}

		redisHooks.clients[i.key] = state
	}

	if state.active.Load() {
		return errors.New("install Redis tracing: client is already instrumented")
	}

	state.mu.Lock()
	state.tracer = otel.GetTracerProvider().Tracer("github.com/redis/go-redis/extra/redisotel")
	state.mu.Unlock()

	if err := state.installNode(i.client); err != nil {
		return fmt.Errorf("install Redis tracing: %w", err)
	}

	state.active.Store(true)
	i.installed = state

	return nil
}

func (i *redisLifecycle) stop(context.Context) error {
	if i.installed != nil {
		i.installed.active.Store(false)
		i.installed = nil
	}

	return nil
}

// Some Redis command names include their second argument in the span name.
func (s *redisInstrumentation) filterCommand(cmd redis.Cmder) bool {
	switch strings.ToLower(cmd.Name()) {
	case "command", "cluster":
		return true
	default:
		return redisotel.DefaultCommandFilter(cmd)
	}
}

func (s *redisInstrumentation) installNode(node *redis.Client) error {
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	for key := range s.nodes {
		if key.Value() == nil {
			delete(s.nodes, key)
		}
	}

	key := weak.Make(node)
	if _, exists := s.nodes[key]; exists {
		return nil
	}

	err := redisotel.InstrumentTracing(
		node,
		redisotel.WithTracerProvider(&redisTracerProvider{state: s}),
		redisotel.WithDBStatement(false),
		redisotel.WithCallerEnabled(false),
		redisotel.WithCommandFilter(
			func(cmd redis.Cmder) bool { return !s.active.Load() || s.filterCommand(cmd) },
		),
		redisotel.WithCommandsFilter(func(cmds []redis.Cmder) bool {
			return !s.active.Load() || slices.ContainsFunc(cmds, s.filterCommand)
		}),
		redisotel.WithDialFilter(true),
	)
	if err != nil {
		return err
	}

	s.nodes[key] = struct{}{}

	return nil
}

type redisTracerProvider struct {
	oteltrace.TracerProvider

	state *redisInstrumentation
}

func (p *redisTracerProvider) Tracer(_ string, _ ...oteltrace.TracerOption) oteltrace.Tracer {
	return &redisTracer{state: p.state}
}

type redisTracer struct {
	oteltrace.Tracer

	state *redisInstrumentation
}

func (t *redisTracer) Start(
	ctx context.Context,
	name string,
	opts ...oteltrace.SpanStartOption,
) (context.Context, oteltrace.Span) {
	t.state.mu.RLock()
	tracer := t.state.tracer
	t.state.mu.RUnlock()

	return tracer.Start(ctx, name, opts...) //nolint:spancheck // The caller owns the returned span.
}

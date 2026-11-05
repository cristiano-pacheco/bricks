# Decode, validate, execute, and respond

Build `POST /orders` with Chi's real listener, JSON decoding, validation at the use-case boundary, an explicitly decorated use case, and HTTP error mapping. The example uses an in-memory catalog so the complete flow needs no external service. Start in an application module that already depends on Bricks.

## 1. Create the configuration

Create `cmd/orders/config/base.yaml`:

```yaml
app:
  http:
    address: 127.0.0.1:8080
    enablemetrics: false
    readtimeout: 15s
    writetimeout: 15s
    idletimeout: 60s
    shutdowntimeout: 10s
```

This is the real server configuration. Parsing, validator rules, decorator dependencies, and response rendering are Go-only here. This manual decorator chain needs no decorator YAML. For additional server fields, load [the complete field reference](../references/http/server/chi/configuration.md) only when changing those settings.

## 2. Create the application

Create `cmd/orders/main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/http/request"
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"github.com/cristiano-pacheco/bricks/pkg/validator"
	"go.uber.org/fx"
)

type OrderInput struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int `json:"quantity" validate:"gte=1,lte=100"`
}

type OrderOutput struct {
	TotalCents int `json:"total_cents"`
}

type Catalog interface {
	Price(context.Context, string) (int, error)
}

type memoryCatalog struct{}

func (memoryCatalog) Price(ctx context.Context, sku string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if sku == "broken" {
		return 0, errors.New("private catalog connection failure")
	}
	if sku != "coffee" {
		return 0, errs.New("PRODUCT_NOT_FOUND", "product not found", http.StatusNotFound, nil)
	}
	return 250, nil
}

type CreateOrder struct {
	validate validator.Validator
	catalog  Catalog
}

func (uc *CreateOrder) Execute(ctx context.Context, input OrderInput) (OrderOutput, error) {
	if err := uc.validate.Validate(input); err != nil {
		return OrderOutput{}, err
	}
	price, err := uc.catalog.Price(ctx, input.SKU)
	if err != nil {
		return OrderOutput{}, err
	}
	return OrderOutput{TotalCents: price * input.Quantity}, nil
}

func provideOrder(v validator.Validator, catalog Catalog, log logger.Logger) ucdecorator.UseCase[OrderInput, OrderOutput] {
	base := &CreateOrder{validate: v, catalog: catalog}
	// Explicit manual wrapping. Metrics and translation are deliberately absent.
	// The tracing wrapper uses the global no-op provider until tracing is installed.
	return ucdecorator.Chain[OrderInput, OrderOutput](base, log, nil, nil, "create_order", "CreateOrder.Execute")
}

func provideLogger(lc fx.Lifecycle) (logger.Logger, error) {
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"orders.log"}
	cfg.ErrorOutputPaths = []string{"orders-errors.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return log.Sync() }})
	return log, nil
}

type OrderRoutes struct {
	create ucdecorator.UseCase[OrderInput, OrderOutput]
	errors response.ErrorHandler
}

func newOrderRoutes(create ucdecorator.UseCase[OrderInput, OrderOutput], handler response.ErrorHandler) chi.Route {
	return &OrderRoutes{create: create, errors: handler}
}

func (routes *OrderRoutes) Setup(server *chi.Server) {
	server.Router().Post("/orders", func(w http.ResponseWriter, r *http.Request) {
		var input OrderInput
		if err := request.ReadJSON(w, r, &input); err != nil {
			routes.errors.Error(w, err)
			return
		}
		output, err := routes.create.Execute(r.Context(), input)
		if err != nil {
			routes.errors.Error(w, err)
			return
		}
		if err := response.JSON(w, http.StatusCreated, output, nil); err != nil {
			// The response may already have started. Do not render a second body.
			slog.ErrorContext(r.Context(), "write order", "error", err)
		}
	})
}

func application() *fx.App {
	return fx.New(
		chi.Module, validator.Module, response.Module,
		fx.Provide(provideLogger, func() Catalog { return memoryCatalog{} }, provideOrder),
		fx.Provide(fx.Annotate(newOrderRoutes, fx.ResultTags(`group:"routes"`))),
	)
}

func run(app *fx.App) error {
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	err := app.Start(startCtx)
	cancelStart()
	if err == nil {
		signal := <-app.Wait()
		if signal.ExitCode != 0 {
			err = fmt.Errorf("application requested exit code %d", signal.ExitCode)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(application()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

The `Catalog` interface is an application port. Its controlled adapter returns 250 cents for `coffee`, a structured 404 for an unknown product, and an unstructured error for the reserved `broken` product. Replace that provider with your own adapter without changing the HTTP route.

`CreateOrder.Execute` validates before accessing the catalog. Successful JSON decoding alone does not validate `quantity` or require `sku`. The route passes `r.Context()` into the decorated use case, so cancellation reaches its adapter.

`provideOrder` explicitly returns the manually wrapped use-case interface. The chain logs outcomes and starts a span using the current global tracing provider. Metrics and translation are deliberately omitted with nil dependencies. No exporter is installed, so this example does not promise visible spans. Installing modules alone would not wrap the use case. For configured factories or optional integrations, load the [manual chain](../references/ucdecorator/manual.md) or [decorator observability recipe](decorator-observability.md) only when needed.

`response.Module` receives the exact `validator.Validator` and Bricks `logger.Logger` dependencies it needs. That logger writes use-case messages to `orders.log`; it is not the optional `*slog.Logger` used by Chi. The logger's stop hook synchronizes its files after server shutdown because the route depends on it. Failures propagate to the runner, which preserves cleanup errors before exiting.

## 3. Run and create an order

```sh
APP_CONFIG_DIR=cmd/orders/config APP_ENV=local go run ./cmd/orders
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"sku":"coffee","quantity":2}'
```

Expect HTTP 201 and `{"data":{"total_cents":500}}`. The success envelope comes from `response.JSON`. This example calculates a quote-like order total. It does not persist an order or claim transactional database behavior.

## 4. Send invalid and failing inputs

Use the same URL and JSON header for these bodies:

| Request body | Expected status | Expected JSON body |
| --- | --- | --- |
| `{"sku":"coffee","quantity":0}` | 422 | `{"error":{"code":"INVALID_ARGUMENT","message":"request has invalid fields","details":[{"field":"quantity","message":"Quantity must be 1 or greater"}]}}` |
| `{"sku":"missing","quantity":1}` | 404 | `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}` |
| `{"sku":"broken","quantity":1}` | 500 | `{"error":{"code":"internal_server_error","message":"Internal server error"}}` |
| `{"sku":"coffee","quantity":1,"extra":true}` | 400 | `{"error":{"code":"BAD_REQUEST","message":"request body contains unknown field \"extra\""}}` |

For example:

```sh
curl -i http://127.0.0.1:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"sku":"broken","quantity":1}'
```

The private catalog connection message must not appear in the 500 response. Structured errors intentionally expose their code and message, so give them client-safe text. The logging decorator records failures separately. The response error handler does not automatically log every unknown use-case error.

For malformed bodies, content-type handling, and byte limits beyond these flow cases, load [JSON parsing](../references/http/request/usage.md). For custom rules, scalar validation, or translated validation messages, load [validation](../references/validator/usage.md). For envelope alternatives and every error-mapping branch, load [HTTP responses](../references/http/response/usage.md). These are optional deeper references, not setup prerequisites.

## 5. Shut down cleanly

Press Ctrl-C. The runner stops the Fx app, Chi drains requests, and the logger synchronizes its output files. Startup and shutdown errors are joined and reported at `main` with a nonzero exit status. A response write error is logged in its callback instead of trying to render a second body.

Create the Go file from step 2 and copy the [configuration](../examples/http-flow/config/base.yaml) into the directory shown above. They need only the application module's Bricks dependency. The request results above describe expected behavior from the implementation, not an executed walkthrough.

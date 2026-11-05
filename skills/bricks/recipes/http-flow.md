# Decode, validate, execute, and respond

Build `POST /orders` with Bricks' Chi server, bounded JSON decoding, use-case validation, and error responses. The use case runs directly. This example has no database or decorators.

Start in an application module that already depends on Bricks. Project conventions may require separate files and packages. The single file below keeps the framework setup visible without prescribing that layout.

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

These are explicit server settings. Validation and response wiring are Go-only. For other server fields, read [server configuration](../references/http/server/chi/configuration.md).

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
    "github.com/cristiano-pacheco/bricks/pkg/validator"
    "go.uber.org/fx"
)

type OrderRequest struct {
    SKU      string `json:"sku"`
    Quantity int    `json:"quantity"`
}

type OrderInput struct {
    SKU      string `validate:"required"`
    Quantity int    `validate:"gte=1,lte=100"`
}

type OrderOutput struct {
    TotalCents int
}

type OrderResponse struct {
    TotalCents int `json:"total_cents"`
}

type Catalog interface {
    Price(context.Context, string) (int, error)
}

type memoryCatalog struct{}

func newMemoryCatalog() *memoryCatalog {
    return &memoryCatalog{}
}

func (catalog *memoryCatalog) Price(ctx context.Context, sku string) (int, error) {
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
    inputValidator validator.Validator
    catalog        Catalog
}

func newCreateOrder(inputValidator validator.Validator, catalog Catalog) *CreateOrder {
    return &CreateOrder{inputValidator: inputValidator, catalog: catalog}
}

func (operation *CreateOrder) Execute(ctx context.Context, input OrderInput) (OrderOutput, error) {
    if err := operation.inputValidator.Validate(input); err != nil {
        return OrderOutput{}, err
    }
    price, err := operation.catalog.Price(ctx, input.SKU)
    if err != nil {
        return OrderOutput{}, err
    }
    return OrderOutput{TotalCents: price * input.Quantity}, nil
}

func provideLogger(lifecycle fx.Lifecycle) (logger.Logger, error) {
    cfg := logger.DefaultConfig()
    cfg.OutputPaths = []string{"orders.log"}
    cfg.ErrorOutputPaths = []string{"orders-errors.log"}
    applicationLogger, err := logger.New(cfg)
    if err != nil {
        return nil, err
    }
    lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
        return applicationLogger.Sync()
    }})
    return applicationLogger, nil
}

type OrderRoutes struct {
    createOrder  *CreateOrder
    errorHandler response.ErrorHandler
}

func newOrderRoutes(createOrder *CreateOrder, errorHandler response.ErrorHandler) *OrderRoutes {
    return &OrderRoutes{createOrder: createOrder, errorHandler: errorHandler}
}

func (routes *OrderRoutes) Setup(server *chi.Server) {
    server.Router().Post("/orders", routes.handleCreate)
}

func (routes *OrderRoutes) handleCreate(w http.ResponseWriter, r *http.Request) {
    var body *OrderRequest
    if err := request.ReadJSON(w, r, &body); err != nil {
        routes.errorHandler.Error(w, err)
        return
    }
    if body == nil {
        routes.errorHandler.Error(w, errs.New("BAD_REQUEST", "request body must be an object", http.StatusBadRequest, nil))
        return
    }
    output, err := routes.createOrder.Execute(r.Context(), OrderInput{SKU: body.SKU, Quantity: body.Quantity})
    if err != nil {
        // This example reports operation failures here. No decorator logs them.
        slog.ErrorContext(r.Context(), "create order", "error", err)
        routes.errorHandler.Error(w, err)
        return
    }
    result := OrderResponse{TotalCents: output.TotalCents}
    if err := response.JSON(w, http.StatusCreated, result, nil); err != nil {
        // The response may have started. Do not render another body.
        slog.ErrorContext(r.Context(), "write order", "error", err)
    }
}

func application() *fx.App {
    return fx.New(
        chi.Module, validator.Module, response.Module,
        fx.Provide(
            provideLogger,
            fx.Annotate(newMemoryCatalog, fx.As(new(Catalog))),
            newCreateOrder,
            fx.Annotate(newOrderRoutes, fx.As(new(chi.Route)), fx.ResultTags(`group:"routes"`)),
        ),
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

The route maps HTTP fields into application input and maps output into a response DTO. `CreateOrder.Execute` validates before calling its catalog. The catalog is an application port. Its controlled adapter returns 250 cents for `coffee`, a safe structured 404 for an unknown product, and an unstructured error for `broken`.

Application constructors return concrete instances. Fx publishes the catalog and routes under their interfaces. `provideLogger` is a bridge to the framework's interface-returning factory. `response.Module` consumes that logger and the validator. The runner preserves shutdown errors before exiting.

This operation calculates an order total. It does not persist an order. No factory, decorator interface, metrics collector, or tracing exporter is required. For requested decoration, select [manual chains](../references/ucdecorator/manual.md) or [decorator factories](../references/ucdecorator/factory.md).

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

Expect HTTP 201 and `{"data":{"total_cents":500}}`.

## 4. Check failures

Send these bodies to the same URL with the same JSON header:

| Body | Expected result |
| --- | --- |
| `{"sku":"coffee","quantity":0}` | 422 with `INVALID_ARGUMENT` and a quantity detail |
| `{"sku":"missing","quantity":1}` | 404 with `PRODUCT_NOT_FOUND` |
| `{"sku":"broken","quantity":1}` | 500 with `internal_server_error` and `Internal server error` |
| `{"sku":"coffee","quantity":1,"extra":true}` | 400 with `BAD_REQUEST` |
| `null` | 400 with `BAD_REQUEST`; the operation does not run |

The private catalog failure must not appear in the 500 response. Structured errors expose their messages, so use client-safe text. The handler reports operation failures through slog. The Bricks logger supplied above handles the error renderer's own failures. Neither is a use-case decorator.

For byte limits, trailing JSON behavior, and content types, read [JSON requests](../references/http/request/usage.md). For custom validation or response formats, load [validation](../references/validator/usage.md) or [responses](../references/http/response/usage.md) only for that change.

## 5. Stop the application

Press Ctrl-C. Fx stops Chi and synchronizes the logger's files. Startup and shutdown failures reach `main` after cleanup. A response-write failure never triggers another response or another operation call.

The [base configuration](../examples/http-flow/config/base.yaml) is also available as an asset. Record the application's actual command, responses, and shutdown result separately.

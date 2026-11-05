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

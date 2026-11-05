package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"go.uber.org/fx"
	"github.com/swaggo/swag"
)

type GreetingRoutes struct{}

func newGreetingRoutes() chi.Route { return &GreetingRoutes{} }

func (r *GreetingRoutes) Setup(server *chi.Server) {
	server.Router().Get("/hello", func(w http.ResponseWriter, request *http.Request) {
		if err := response.JSON(w, http.StatusOK, map[string]string{"message": "hello"}, nil); err != nil {
			slog.ErrorContext(request.Context(), "write greeting", "error", err)
		}
	})
}

type swaggerDocument struct{}

func (swaggerDocument) ReadDoc() string {
	return `{"swagger":"2.0","info":{"title":"Greeting API","version":"1.0"},"basePath":"/","paths":{"/hello":{"get":{"responses":{"200":{"description":"Greeting"}}}}}}`
}

func application() *fx.App {
	swag.Register("swagger", swaggerDocument{})
	return fx.New(
		chi.Module,
		fx.Provide(fx.Annotate(newGreetingRoutes, fx.ResultTags(`group:"routes"`))),
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

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
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/cristiano-pacheco/bricks/pkg/paginator"
	"go.uber.org/fx"
)

type ListOutput struct {
	Items []string `json:"items"`
	Pagination paginator.Metadata `json:"pagination"`
}

type ProductRoutes struct{}

func newProductRoutes() chi.Route { return &ProductRoutes{} }

func (routes *ProductRoutes) Setup(server *chi.Server) {
	handler := response.NewErrorHandler(nil, nil)
	server.Router().Get("/products", func(w http.ResponseWriter, r *http.Request) {
		params, err := paginator.ParseQueryParams(r.URL.Query(), 1, 2)
		if err != nil {
			handler.Error(w, errs.New("BAD_REQUEST", err.Error(), http.StatusBadRequest, nil))
			return
		}
		// This application rejects excessive page numbers before multiplying offsets.
		if params.Page > 1_000_000 {
			handler.Error(w, errs.New("BAD_REQUEST", "page exceeds application limit", http.StatusBadRequest, nil))
			return
		}
		params = paginator.NormalizeParams(params.Page, params.PerPage, 1, 2, 2)
		all := []string{"coffee", "tea", "cocoa"}
		start := params.Offset()
		if start > len(all) {
			start = len(all)
		}
		end := start + params.Limit()
		if end > len(all) {
			end = len(all)
		}
		output := ListOutput{
			Items: all[start:end],
			Pagination: paginator.Metadata{TotalCount: int64(len(all)), Page: params.Page,
				PerPage: params.PerPage, TotalPages: paginator.TotalPages(int64(len(all)), params.PerPage)},
		}
		if err := response.JSON(w, http.StatusOK, output, nil); err != nil {
			slog.ErrorContext(r.Context(), "write products", "error", err)
		}
	})
}

func application() *fx.App {
	return fx.New(chi.Module, fx.Provide(fx.Annotate(newProductRoutes, fx.ResultTags(`group:"routes"`))))
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

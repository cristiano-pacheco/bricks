# Return a paginated list with response metadata

Serve `GET /products` with strict query parsing, an explicit application page-size cap, bounded offsets, and metadata inside the JSON success envelope. This example uses three in-memory products so it needs no database.

## 1. Create the server configuration

In an application module that already depends on Bricks, create `cmd/products/config/base.yaml`:

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

Pagination policy and metadata are code-only. There are no YAML page-size keys in `paginator`. The YAML above configures only the Chi listener and its lifecycle. For optional server fields, load [the full configuration reference](../references/http/server/chi/configuration.md).

## 2. Create the application

Create `cmd/products/main.go`:

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
```

`ParseQueryParams` rejects supplied zero, negative, noninteger, and overflowing integer values. Defaults apply to blank or absent values. The next normalization step caps valid sizes at two. It is an application policy, not behavior supplied by the strict parser. This example also limits page numbers before calculating offsets to avoid unchecked integer multiplication.

The slice adapter applies `Offset()` and `Limit()`. A database adapter would use those same normalized values for its query and obtain an independent total count without the page limit. Metadata uses that total count, not the number of rows in this page. An out-of-range page returns an empty list and preserves the requested page number.

`response.NewErrorHandler(nil, nil)` is a supported direct-construction fallback here because no validation errors need translations and no Bricks logger is required. It uses slog if writing an error response itself fails. Do not assume this removes the dependencies of `response.Module` in a different application.

## 3. Start and request pages

```sh
APP_CONFIG_DIR=cmd/products/config APP_ENV=local go run ./cmd/products
```

In another terminal:

```sh
curl -i 'http://127.0.0.1:8080/products?page=2&per_page=20'
curl -i 'http://127.0.0.1:8080/products?page=1&per_page=0'
curl -i 'http://127.0.0.1:8080/products?page=3&per_page=2'
```

The first request caps the size to two, computes offset two, and returns HTTP 200:

```json
{"data":{"items":["cocoa"],"pagination":{"total_count":3,"page":2,"per_page":2,"total_pages":2}}}
```

The second request returns HTTP 400:

```json
{"error":{"code":"BAD_REQUEST","message":"invalid per_page"}}
```

The third request returns HTTP 200:

```json
{"data":{"items":[],"pagination":{"total_count":3,"page":3,"per_page":2,"total_pages":2}}}
```

Without query parameters, the defaults are page one and size two, so the first page contains `coffee` and `tea`. These bodies are expected results, not an execution record.

## 4. Stop the application

Press Ctrl-C. The runner waits for Chi shutdown and reports lifecycle errors only after cleanup. No external client or storage cleanup is needed.

Copy the complete [program](../examples/chi-pagination/main.go) and [YAML](../examples/chi-pagination/config/base.yaml) when you want the runnable version. The setup above does not require other guides.

## Supported alternatives

Choose your own positive defaults and cap. Set `maxPerPage` to zero or a negative number only when you deliberately want no cap. If you accept unchecked internal pagination values instead of an HTTP query, normalization can supply fallbacks, but it does not validate those fallback arguments or protect offset multiplication.

`paginator.Metadata` is an application payload. Bricks does not automatically insert it into `response.JSON`. The explicit `ListOutput` puts it under `data.pagination`; `response.JSONRaw` would omit only the outer `data` envelope. For other API contracts, use [HTTP responses](../references/http/response/usage.md). For scalar filters, overflow behavior, aliases, and helper details, use [pagination usage](../references/paginator/usage.md).

# Expose an existing operation over HTTP

Use this recipe when the application already has an order query and needs a Bricks Chi endpoint. Reuse its server, input validator, logger, and `response.Module`. Do not create another server or copy the operation's policy into the handler.

The example assumes the application provides `usecase.OrderGetUseCase.Execute(context.Context, OrderGetInput) (OrderGetOutput, error)`. Its input has `ID` and `ActorID`; its output has `ID` and `Status`. Replace the example module path and error catalog with the actual application's contracts. For a complete new application without those prerequisites, use [HTTP flow](http-flow.md).

## 1. Add the HTTP response DTO

Create `internal/modules/order/http/dto/order_get_dto.go`:

```go
package dto

type OrderGetResponse struct {
    ID uint64 `json:"id"`
    Status string `json:"status"`
}
```

The DTO owns JSON names. Application Output remains independent of HTTP.

## 2. Add the handler and route

Create `internal/modules/order/http/chi/handler/order_handler.go`:

```go
package handler

import (
    "context"
    "log/slog"
    "net/http"
    "strconv"

    "example.com/project/internal/modules/order/errs"
    httpdto "example.com/project/internal/modules/order/http/dto"
    "example.com/project/internal/modules/order/usecase"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
    brickschi "github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
    "github.com/go-chi/chi/v5"
)

type ActorID func(context.Context) (uint64, bool)

type OrderHandler struct {
    orderGetUseCase *usecase.OrderGetUseCase
    actorID ActorID
    errorHandler response.ErrorHandler
}

func NewOrderHandler(
    orderGetUseCase *usecase.OrderGetUseCase,
    actorID ActorID,
    errorHandler response.ErrorHandler,
) *OrderHandler {
    return &OrderHandler{
        orderGetUseCase: orderGetUseCase,
        actorID: actorID,
        errorHandler: errorHandler,
    }
}

func (handler *OrderHandler) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
    actorID, authenticated := handler.actorID(r.Context())
    if !authenticated || actorID == 0 {
        handler.errorHandler.Error(w, errs.ErrUnauthenticated)
        return
    }
    id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
    if err != nil || id == 0 {
        handler.errorHandler.Error(w, errs.ErrInvalidInput)
        return
    }
    output, err := handler.orderGetUseCase.Execute(r.Context(), usecase.OrderGetInput{ID: id, ActorID: actorID})
    if err != nil {
        handler.errorHandler.Error(w, err)
        return
    }
    result := httpdto.OrderGetResponse{ID: output.ID, Status: output.Status}
    if err := response.JSON(w, http.StatusOK, result, nil); err != nil {
        slog.ErrorContext(r.Context(), "write order response", "error", err)
    }
}

type OrderRoutes struct {
    orderHandler *OrderHandler
}

func NewOrderRoutes(orderHandler *OrderHandler) *OrderRoutes {
    return &OrderRoutes{orderHandler: orderHandler}
}

func (routes *OrderRoutes) Setup(server *brickschi.Server) {
    server.Router().Get("/api/v1/orders/{id}", routes.orderHandler.HandleGetOrder)
}
```

`ActorID` must use the application's trusted middleware or session integration. Never read it from a query or JSON field or fabricate a default actor. Apply the existing authentication middleware to this route group. The operation still checks business access.

The catalog's known rejections must follow the application's approved Bricks-compatible error mapping. Bare Go sentinels do not become typed HTTP statuses automatically. Preserve the existing reporting owner for unknown operation failures. `response.ErrorHandler` logs its own rendering failures, not every original application error.

## 3. Register the contribution once

Add this fragment to the owner module's Fx options. Import the handler package, `go.uber.org/fx`, and Bricks Chi as `brickschi`:

```go
fx.Provide(
    handler.NewOrderHandler,
    fx.Annotate(handler.NewOrderRoutes,
        fx.As(new(brickschi.Route)),
        fx.ResultTags(`group:"routes"`)),
)
```

Supply the real `handler.ActorID` function at the process owner. The existing process must already provide the concrete operation, Bricks validator and logger, `response.Module`, and Chi server. The route group reaches the handler and its dependencies. Do not register another copy of those modules to add one endpoint.

The application constructors above return concrete pointers. Fx exposes `OrderRoutes` as the framework route interface. A project with separate handler and router packages can split the code without changing those contracts.

## 4. Run the existing application

Use its normal server command and configuration. Request an existing order through the application's actual authentication mechanism:

```sh
curl -i http://127.0.0.1:8080/api/v1/orders/42
```

The URL is an example local address. Without trusted authentication, expect the application's 401 result. With access to order 42, expect HTTP 200 and a single data envelope such as `{"data":{"id":42,"status":"pending"}}`. Check invalid IDs, denied access, absent records, and safe unexpected errors under the application's published policy.

These are expected outcomes, not execution evidence. This recipe changes no database or use-case policy. For a body-based endpoint, use [HTTP flow](http-flow.md). For different envelopes or complete error classification, read [responses](../references/http/response/usage.md) only when needed.

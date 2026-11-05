package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"go.uber.org/fx"
)

//go:embed locales/*.json
var files embed.FS

type QuoteInput struct {
	Quantity int
}

type QuoteUseCase struct{}

func (*QuoteUseCase) Execute(ctx context.Context, input QuoteInput) (int, error) {
	if input.Quantity < 1 {
		return 0, errs.New("QUOTE_QUANTITY", "Quantity must be positive.", http.StatusBadRequest, nil)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return input.Quantity * 125, nil
}

func provideLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "locales")
	return locale.New(root), err
}

func newLogger() (logger.Logger, error) {
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"i18n.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	return log, nil
}

func provideQuote(factory *ucdecorator.Factory) ucdecorator.UseCase[QuoteInput, int] {
	return ucdecorator.Wrap[QuoteInput, int](factory, &QuoteUseCase{})
}

func run() (result error) {
	log, err := newLogger()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	count := 2
	if len(os.Args) > 1 {
		var err error
		count, err = strconv.Atoi(os.Args[1])
		if err != nil {
			return err
		}
	}
	var quote ucdecorator.UseCase[QuoteInput, int]
	app := fx.New(
		brickstrace.Module,
		i18n.Module,
		metrics.Module,
		ucdecorator.Module,
		fx.Provide(func() logger.Logger { return log }, provideQuote,
			fx.Annotate(provideLocales, fx.ResultTags(`group:"locale_filesystems"`)),
		),
		fx.Populate(&quote),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		var total int
		total, err = quote.Execute(context.Background(), QuoteInput{Quantity: count})
		if err == nil {
			fmt.Printf("total_cents=%d\n", total)
		}
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

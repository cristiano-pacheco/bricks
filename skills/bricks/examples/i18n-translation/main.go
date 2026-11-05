package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/i18n"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"go.uber.org/fx"
)

//go:embed locales/*.json extra/*.json
var files embed.FS

func provideLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "locales")
	return locale.New(root), err
}

func provideExtraLocales() (locale.FileSystem, error) {
	root, err := fs.Sub(files, "extra")
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

func printMessages(translator ports.TranslationService) {
	fmt.Println(translator.Translate("catalog", "title"))
	fmt.Println(translator.TranslateWithData("catalog", "welcome", map[string]string{"Name": "Ana"}))
	fmt.Println(translator.Translate("catalog", "only_en"))
	fmt.Println(translator.Translate("catalog", "unknown"))
	fmt.Println(translator.Translate("checkout", "paid"))
}

func run() (result error) {
	log, err := newLogger()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	var translator ports.TranslationService
	app := fx.New(
		i18n.Module,
		fx.Provide(func() logger.Logger { return log },
			fx.Annotate(provideLocales, fx.ResultTags(`group:"locale_filesystems"`)),
			fx.Annotate(provideExtraLocales, fx.ResultTags(`group:"locale_filesystems"`)),
		),
		fx.Populate(&translator),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
	err = app.Start(startCtx)
	cancelStart()
	if err == nil {
		printMessages(translator)
	}
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStop()
	return errors.Join(err, app.Stop(stopCtx))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

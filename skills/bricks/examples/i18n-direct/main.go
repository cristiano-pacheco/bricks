package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

//go:embed locales/*.json
var files embed.FS

func run() (result error) {
	cfg := logger.DefaultConfig()
	cfg.OutputPaths = []string{"i18n.log"}
	log, err := logger.New(cfg)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Sync()) }()
	root, err := fs.Sub(files, "locales")
	if err != nil {
		return err
	}
	loader := service.NewLocaleLoaderService(log, []locale.FileSystem{locale.New(root)})
	translator, err := service.NewTranslationService(loader, config.Config{Language: "pt-BR"}, log)
	if err != nil {
		return err
	}
	fmt.Println(translator.TranslateWithData("catalog", "welcome", map[string]string{"Name": "Ana"}))
	errorTranslator := service.NewErrorTranslatorService(translator)
	original := errs.New("QUOTE_QUANTITY", "Quantity must be positive.", http.StatusBadRequest, nil)
	fmt.Println(errorTranslator.TranslateError(original))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

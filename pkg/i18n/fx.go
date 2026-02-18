package i18n

import (
	bricksconfig "github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"i18n",
	bricksconfig.Provide[config.Config]("app.i18n"),
	fx.Provide(
		func(c bricksconfig.Config[config.Config]) config.Config {
			return c.Get()
		},
		fx.Annotate(
			service.NewLocaleLoaderService,
			fx.As(new(ports.LocaleLoaderService)),
		),
		fx.Annotate(
			service.NewTranslationService,
			fx.As(new(ports.TranslationService)),
		),
		fx.Annotate(
			service.NewErrorTranslatorService,
			fx.As(new(ports.ErrorTranslatorService), new(ucdecorator.ErrorTranslator)),
		),
	),
)

package database

import (
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

// Module loads app.database and provides a lifecycle-managed *gorm.DB.
var Module = fx.Module("database",
	config.Provide[Config]("app.database"),
	fx.Provide(newFromConfig),
)

func newFromConfig(cfg config.Config[Config], lc fx.Lifecycle) (*gorm.DB, error) {
	return NewWithLifecycle(cfg.Get(), lc)
}

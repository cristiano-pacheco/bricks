package sqlite

import (
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
)

// Module provides a GORM SQLite database with Fx lifecycle management.
// It loads Config from the app.sqlite configuration path.
var Module = fx.Module(
	"database-sqlite",
	config.Provide[Config]("app.sqlite"),
	fx.Provide(NewWithLifecycle),
)

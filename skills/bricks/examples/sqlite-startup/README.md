# Sqlite startup assets

Follow [the setup guide](../../recipes/sqlite-migrations.md) and copy the required assets into your application. This directory contains configuration or data, not an executable Go application. Create the Go files from the inline instructions in that guide.

For Bricks configuration, keep the required `config/base.yaml`. Select an alternative overlay with `APP_ENV` or copy its values into the base file as the guide directs.

Keep each SQL migration filename and its matching up/down pair. Copy `migrations/` beside the application package that embeds or loads it.

## Files

- [config/base.yaml](config/base.yaml)
- [migrations/000001_create_items.down.sql](migrations/000001_create_items.down.sql)
- [migrations/000001_create_items.up.sql](migrations/000001_create_items.up.sql)

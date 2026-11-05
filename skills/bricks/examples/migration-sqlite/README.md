# Migration sqlite assets

Follow [the setup guide](../../references/migration/sqlite.md) and copy the required assets into your application. This directory contains configuration or data, not an executable Go application. Create the Go files from the inline instructions in that guide.

Keep each SQL migration filename and its matching up/down pair. Copy `migrations/` beside the application package that embeds or loads it.

## Files

- [migrations/000001_create_accounts.down.sql](migrations/000001_create_accounts.down.sql)
- [migrations/000001_create_accounts.up.sql](migrations/000001_create_accounts.up.sql)

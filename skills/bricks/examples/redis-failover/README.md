# Redis failover assets

Follow [the setup guide](../../references/redis/failover.md) and copy the required assets into your application. This directory contains configuration or data, not an executable Go application. Create the Go files from the inline instructions in that guide.

For Bricks configuration, keep the required `config/base.yaml`. Select an alternative overlay with `APP_ENV` or copy its values into the base file as the guide directs.

## Files

- [config/base.yaml](config/base.yaml)
- [config/minimal.yaml](config/minimal.yaml)
- [config/tls.yaml](config/tls.yaml)

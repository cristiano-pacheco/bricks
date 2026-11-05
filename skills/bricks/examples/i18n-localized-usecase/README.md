# I18n localized usecase assets

Follow [the setup guide](../../recipes/localized-usecases.md) and copy the required assets into your application. This directory contains configuration or data, not an executable Go application. Create the Go files from the inline instructions in that guide.

For Bricks configuration, keep the required `config/base.yaml`. Select an alternative overlay with `APP_ENV` or copy its values into the base file as the guide directs.

Preserve the locale directory names used by the guide. The i18n loader needs `<language>.json` at each contributed filesystem root; embedded files must be rooted with `fs.Sub` as shown in the guide.

## Files

- [config/base.yaml](config/base.yaml)
- [config/export.yaml](config/export.yaml)
- [config/untranslated.yaml](config/untranslated.yaml)
- [locales/en.json](locales/en.json)
- [locales/pt-BR.json](locales/pt-BR.json)

# I18n direct assets

Follow [the setup guide](../../references/i18n/service/usage.md) and copy the required assets into your application. This directory contains configuration or data, not an executable Go application. Create the Go files from the inline instructions in that guide.

Preserve the locale directory names used by the guide. The i18n loader needs `<language>.json` at each contributed filesystem root; embedded files must be rooted with `fs.Sub` as shown in the guide.

## Files

- [locales/en.json](locales/en.json)
- [locales/pt-BR.json](locales/pt-BR.json)

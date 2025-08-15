# Translation Marking

- All user-visible strings must pass through 'translate.From()' in the codebase, which maps to `golang.org/x/text/message.Printer.Sprintf()`

Example:

```
  f := translate.From

  err = fmt.Error(f("fall down detected"))

  fmt.Print(f("user %v won %d tokens!", user_id, token_count))
```

# Translation Workflow

1. Add new locale xx-YY to internal/translate/translate.go's 'go:generate'
2. Run `go generate ./...`
3. Send `internal/translate/locales/xx-YY/out.gotext.json` to translator.
4. Copy translation to `internal/translate/locales/xx-YY/messages.gotext.json`
5. Run `go generate ./...` again to update `internal/translate/catalog.go`
6. Commit.

# kotoshu-go

Go client for the [Kotoshu](https://github.com/kotoshu/kotoshu) HTTP spell-check API.

## Install

```bash
go get github.com/kotoshu/kotoshu-go
```

## Quick start

```go
import kotoshu "github.com/kotoshu/kotoshu-go"

ctx := context.Background()
client, _ := kotoshu.New("http://localhost:9292")

// Check a document
result, _ := client.Check(ctx, "helo wrold", &kotoshu.CheckOptions{Language: "en"})
for _, e := range result.Errors {
    fmt.Println(e.Word, "->", topThree(e.Suggestions))
}

// Suggestions for one word
sugs, _ := client.Suggest(ctx, "helo", &kotoshu.SuggestOptions{Language: "en", Max: 3})

// Language detection
det, _ := client.Detect(ctx, "bonjour le monde")
fmt.Println(det.Language, det.Confidence)

// Quick predicate
ok, _ := client.Correct(ctx, "hello", &kotoshu.CheckOptions{Language: "en"})
```

## Reference

### `New(baseURL string, opts ...) (*Client, error)`

Options:
- `WithLanguage("en")` — default language for all calls
- `WithTimeout(30 * time.Second)` — HTTP timeout
- `WithHTTPClient(client)` — inject a custom `*http.Client`

### Methods (all take `context.Context` first)

| Method | Returns |
|---|---|
| `Health(ctx)` | `*Health, error` |
| `Languages(ctx)` | `[]string, error` |
| `Check(ctx, text, *CheckOptions)` | `*DocumentResult, error` |
| `Suggest(ctx, word, *SuggestOptions)` | `[]Suggestion, error` |
| `Detect(ctx, text)` | `*Detection, error` |
| `Correct(ctx, word, *CheckOptions)` | `bool, error` |

### Errors

Server errors are returned as `*kotoshu.APIError` with `.Code` and `.Message`.
`kotoshu.IsResourceNotSetup(err)` checks for the 422 case.

## License

BSD-2-Clause, same as Kotoshu.

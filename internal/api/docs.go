// internal/api/docs.go
package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed openapi.yaml
var openapiYAML []byte

// swagger-ui files are copied from web/node_modules/swagger-ui-dist/ by
// `make ui-build`. They are embedded here so no external network calls are
// made when serving the docs.
//
//go:embed all:swagger-ui
var swaggerUIFS embed.FS

const swaggerHTML = `<!doctype html>
<html>
<head>
  <title>Galactica API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="/api/docs/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="/api/docs/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: '/api/docs/openapi.yaml',
  dom_id: '#swagger-ui',
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
  layout: 'BaseLayout'
})
</script>
</body>
</html>`

// DocsHandler serves Swagger UI at /api/docs using locally embedded assets.
// No external scripts — all files served from the binary itself.
func DocsHandler() http.Handler {
	sub, err := fs.Sub(swaggerUIFS, "swagger-ui")
	if err != nil {
		panic("swagger-ui embed: " + err.Error())
	}
	staticFiles := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/docs")

		switch path {
		case "", "/", "/index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(swaggerHTML)) //nolint:errcheck
		case "/openapi.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			w.Write(openapiYAML) //nolint:errcheck
		default:
			r = r.Clone(r.Context())
			r.URL.Path = path
			staticFiles.ServeHTTP(w, r)
		}
	})
}

// internal/server/embed.go
package server

import "embed"

// webDist holds the built React SPA.
// Vite outputs to internal/server/dist/ so this path is valid for go:embed.
//
//go:embed all:dist
var webDist embed.FS

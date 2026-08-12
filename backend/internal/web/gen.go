// Package web exposes build-time, content-addressed static assets and build
// metadata for the rendered HTML shell.
//
// Regenerate the embedded CSS after editing templates/post.css:
//
//	go generate ./internal/web
package web

//go:generate go run ../../cmd/buildcss

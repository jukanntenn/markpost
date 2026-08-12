// Package markpost is the backend module root. This file exists only to host
// the go:generate directive that regenerates the Swagger docs; it must live at
// the module root so swag resolves -g/-o paths relative to it.
package markpost

//go:generate go tool swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

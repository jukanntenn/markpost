package web

// buildID identifies the running build. It is retained purely as defense against
// a future hot-reload of templates without a process restart (currently
// impossible): the render-cache keys namespace it so a build rotation clears the
// whole cache automatically. The default value is a stable sentinel; the real
// value is injected at link time via -ldflags "-X markpost/internal/web.buildID=<hash>".
var buildID = "dev"

// BuildID returns the current build identifier.
func BuildID() string { return buildID }

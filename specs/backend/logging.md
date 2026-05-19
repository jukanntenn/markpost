# Backend Logging

## Approach

The backend uses Go's standard library `log/slog` package for structured logging. All logging goes to stdout/stderr.

## When to Log

Log at startup for significant lifecycle events:

- Configuration loaded
- Database initialized
- Admin user created (`log.Printf("initialized user: %s", username)`)
- Server starting (`log.Println("Server starting...")`)
- Server listening address (`log.Println("Visit http://...")`)
- Post QID migration (`log.Printf("migrated %d post qids with p- prefix", n)`)

Log on unexpected errors that bubble up without causing a crash:

- Render post errors (`slog.Error("render post failed", "error", err, "path", path)`)
- Unexpected error types in `apierr.RespondError` (`log.Printf("unexpected error: %v", err)`)
- Unknown service error codes (`log.Printf("unknown service error code: %s ...", ...)`)

## What Not to Log

Never log sensitive data:

- Passwords (plain or hashed)
- JWT tokens (access or refresh)
- OAuth client secrets
- Post key values (in production logs)
- Full request bodies that may contain user content

## Error Logging

Service-layer errors are not logged individually — they are returned to the handler layer which converts them to HTTP responses. The `apierr` package logs only truly unexpected conditions:

1. Non-`service.Error` types passed to `RespondError` — logged as "unexpected error"
2. Unrecognized error codes — logged with code, description, and wrapped error

## Fatal Logging

`log.Fatalf` is used during startup when the server cannot continue:

- Config file loading failure
- Database connection failure
- Admin user initialization failure
- Trusted proxy configuration failure
- Server bind failure

These calls exit the process immediately after printing the message.

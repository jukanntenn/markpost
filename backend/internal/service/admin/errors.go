package admin

// Admin-domain error codes. The admin service is a cross-aggregate read-only
// view and currently relies on the shared service error codes
// (ErrNotFound/ErrForbidden/ErrInternal); this file is reserved for any
// admin-specific codes added in the future, per error-handling.md's
// "域专属码分文件" principle.

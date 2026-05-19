// KEEP IN SYNC with backend/internal/service/errors.go
export const ApiErrorCodes = {
  InvalidCredentials: "invalid_credentials",
  InvalidPassword: "invalid_password",
  Unauthorized: "unauthorized",
  Internal: "internal",
  Validation: "validation",
  NotFound: "not_found",
  FailedGetUser: "failed_get_user",
  MissingAuthorizationHeader: "missing_authorization_header",
  InvalidToken: "invalid_token",
  InvalidPostKey: "invalid_post_key",
  UserDisabled: "user_disabled",
  Forbidden: "forbidden",
  RateLimited: "rate_limited",
  InvalidRequest: "invalid_request",
  Required: "required",
  MinLength: "min_length",
  FieldViolation: "field_violation",
} as const;

export type ApiErrorCode = (typeof ApiErrorCodes)[keyof typeof ApiErrorCodes];

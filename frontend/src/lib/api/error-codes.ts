// KEEP IN SYNC with backend/internal/service/**/errors.go. Per-domain codes
// (auth/post/delivery/admin) live in their own files; shared codes in
// service/errors.go.
export const ApiErrorCodes = {
  // shared
  Internal: 'internal',
  Validation: 'validation',
  InvalidRequest: 'invalid_request',
  NotFound: 'not_found',
  Unauthorized: 'unauthorized',
  Forbidden: 'forbidden',
  Conflict: 'conflict',
  RateLimited: 'rate_limited',
  // field-validation
  Required: 'required',
  MinLength: 'min_length',
  MaxLength: 'max_length',
  Length: 'length',
  InvalidEmail: 'invalid_email',
  NotOneOf: 'not_one_of',
  FieldViolation: 'field_violation',
  // auth domain
  InvalidCredentials: 'invalid_credentials',
  InvalidPassword: 'invalid_password',
  UserDisabled: 'user_disabled',
  InvalidToken: 'invalid_token',
  InvalidPostKey: 'invalid_post_key',
  MissingState: 'missing_state',
  MissingCode: 'missing_code',
  InvalidState: 'invalid_state',
  OAuthExchangeFailed: 'oauth_exchange_failed',
  GitHubUserFetchFailed: 'github_user_fetch_failed',
  PasswordTooShort: 'password_too_short',
  PasswordTooLong: 'password_too_long',
  // C2.1: account-level login lockout
  AccountLocked: 'account_locked',
  // K.7 D3-3: governance self/last-admin guards
  SelfForbidden: 'self_forbidden',
  LastAdmin: 'last_admin',
  // post domain
  TitleTooLong: 'title_too_long',
  BodyTooLarge: 'body_too_large',
  // delivery domain
  UnsupportedChannelKind: 'unsupported_channel_kind',
  // I.3 SSRF gate
  WebhookURLForbidden: 'webhook_url_forbidden',
  // B1.9: frontend-constructed codes (base.ts, non-backend)
  NetworkError: 'network_error',
  Timeout: 'timeout',
  ParseError: 'parse_error',
} as const

export type ApiErrorCode = (typeof ApiErrorCodes)[keyof typeof ApiErrorCodes]

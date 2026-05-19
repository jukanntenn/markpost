export interface FieldError {
  field?: string;
  code: string;
  message: string;
}

export interface ApiErrorResponse {
  code?: string;
  message?: string;
  errors?: FieldError[];
}

export interface PasswordChangeValues {
  newPassword: string;
  confirmPassword: string;
}

export function validatePasswordChange(
  values: PasswordChangeValues,
  messages: { notMatch: string; minLength: string },
): string | null {
  if (values.newPassword !== values.confirmPassword) {
    return messages.notMatch;
  }
  if (values.newPassword.length < 6) {
    return messages.minLength;
  }
  return null;
}

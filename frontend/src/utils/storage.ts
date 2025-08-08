// Storage keys
const TOKEN_KEY = "auth_token";
const USER_KEY = "auth_user";

// Token storage functions
export const saveToken = (token: string): void => {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch (error) {
    console.error("Failed to save token to localStorage:", error);
    throw new Error("Failed to save authentication token");
  }
};

export const getToken = (): string | null => {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch (error) {
    console.error("Failed to get token from localStorage:", error);
    return null;
  }
};

export const removeToken = (): void => {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch (error) {
    console.error("Failed to remove token from localStorage:", error);
  }
};

export const isTokenValid = (): boolean => {
  const token = getToken();
  if (!token) return false;

  try {
    // Basic JWT token validation (check if it's not expired)
    const payload = JSON.parse(atob(token.split(".")[1]));
    const currentTime = Math.floor(Date.now() / 1000);
    return payload.exp > currentTime;
  } catch (error) {
    console.error("Failed to validate token:", error);
    return false;
  }
};

// User storage functions
export const saveUser = (user: unknown): void => {
  try {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  } catch (error) {
    console.error("Failed to save user to localStorage:", error);
    throw new Error("Failed to save user information");
  }
};

export const getUser = (): unknown | null => {
  try {
    const userStr = localStorage.getItem(USER_KEY);
    return userStr ? JSON.parse(userStr) : null;
  } catch (error) {
    console.error("Failed to get user from localStorage:", error);
    return null;
  }
};

export const removeUser = (): void => {
  try {
    localStorage.removeItem(USER_KEY);
  } catch (error) {
    console.error("Failed to remove user from localStorage:", error);
  }
};

// Clear all auth data
export const clearAuthData = (): void => {
  removeToken();
  removeUser();
};

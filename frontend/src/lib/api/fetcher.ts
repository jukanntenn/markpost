import { storage } from "@/utils/storage";
import type { UserInfo } from "@/components/UserInfoContext";

export async function authFetcher<T>(url: string): Promise<T> {
  const loginData = storage.get<UserInfo>("login");
  const token = loginData?.access_token;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    headers,
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  return response.json();
}

export async function authPoster<T>(url: string, data: unknown): Promise<T> {
  const loginData = storage.get<UserInfo>("login");
  const token = loginData?.access_token;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  return response.json();
}

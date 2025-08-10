import axios from "axios";
import { get } from "./storage";

export const anno = axios.create({
  baseURL: import.meta.env.VITE_BASE_URL,
  timeout: 10000,
});

export const auth = axios.create({
  baseURL: import.meta.env.VITE_BASE_URL,
  timeout: 10000,
});

// Add request interceptor to include JWT token
auth.interceptors.request.use(
  (config) => {
    try {
      const loginData = get<any>("login");
      const accessToken = loginData?.access_token;

      if (accessToken) {
        config.headers.Authorization = `Bearer ${accessToken}`;
      }
    } catch (error) {
      console.error("Error reading login data from storage:", error);
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

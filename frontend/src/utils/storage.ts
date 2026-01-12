const prefix = (import.meta.env.VITE_STORAGE_PREFIX as string | undefined) || "markpost_dev_";

export const get = <T>(key: string, storage: Storage = localStorage): T => {
  const json = storage.getItem(prefix + key);
  try {
    return JSON.parse(json as string) as T;
  } catch {
    return json as unknown as T;
  }
};

export const set = (
  key: string,
  value: unknown,
  storage: Storage = localStorage
): void => {
  storage.setItem(prefix + key, JSON.stringify(value));
};

export const remove = (key: string, storage: Storage = localStorage): void => {
  storage.removeItem(prefix + key);
};

const prefix = import.meta.env.VITE_STORAGE_PREFIX as string;

export const get = <T>(key: string, storage: any = localStorage): T => {
  const json = storage.getItem(prefix + key);
  try {
    return JSON.parse(json as string);
  } catch {
    return json as any;
  }
};

export const set = (
  key: string,
  value: unknown,
  storage: any = localStorage
): void => {
  storage.setItem(prefix + key, JSON.stringify(value));
};

export const remove = (key: string, storage: any = localStorage): void => {
  storage.removeItem(prefix + key);
};

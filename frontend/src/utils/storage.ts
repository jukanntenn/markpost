const prefix = "markpost_";

export const get = <T>(key: string, storage?: Storage): T | null => {
  if (typeof window === "undefined") return null;
  
  const s = storage || localStorage;
  const json = s.getItem(prefix + key);
  if (!json) return null;
  try {
    return JSON.parse(json) as T;
  } catch {
    return json as unknown as T;
  }
};

export const set = (
  key: string,
  value: unknown,
  storage?: Storage
): void => {
  if (typeof window === "undefined") return;
  
  const s = storage || localStorage;
  s.setItem(prefix + key, JSON.stringify(value));
};

export const remove = (key: string, storage?: Storage): void => {
  if (typeof window === "undefined") return;
  
  const s = storage || localStorage;
  s.removeItem(prefix + key);
};

export const storage = {
  get,
  set,
  remove,
};

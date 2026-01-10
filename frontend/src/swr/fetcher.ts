import { auth, anno } from "../utils/api";

export const authFetcher = (url: string) => auth.get(url).then((res) => res.data);

export const annoFetcher = (url: string) => anno.get(url).then((res) => res.data);

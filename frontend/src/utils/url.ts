export function buildPostUrl(qid: string): string {
  const baseUrl = import.meta.env.VITE_BASE_URL?.trim() || "";

  if (!baseUrl || !baseUrl.startsWith("http")) {
    return `/${qid}`;
  }

  const normalizedBaseUrl = baseUrl.replace(/\/+$/, "");
  return `${normalizedBaseUrl}/${qid}`;
}

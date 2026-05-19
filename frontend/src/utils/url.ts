export function buildPostUrl(qid: string): string {
  return `/${qid}`;
}

export function buildFullPostUrl(qid: string): string {
  if (typeof window === "undefined") return "";
  return `${window.location.origin}${buildPostUrl(qid)}`;
}

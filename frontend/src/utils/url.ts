export function buildPostUrl(qid: string): string {
  const serverPort = process.env.MARKPOST_SERVER__PORT || "7330";
  const baseUrl = `http://localhost:${serverPort}`;
  return `${baseUrl}/${qid}`;
}

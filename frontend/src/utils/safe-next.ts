// K.3 intended-URL 安全：open redirect 防护。
// 只接受站内相对路径，拒绝 //host 与外部 URL。
export function safeNext(next: string | null | undefined): string {
  if (!next) return '/dashboard'
  if (!next.startsWith('/') || next.startsWith('//')) return '/dashboard'
  return next
}

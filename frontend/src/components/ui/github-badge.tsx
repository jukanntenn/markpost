import { GithubIcon } from 'lucide-react'

// Icon-only marker beside usernames for GitHub-OAuth accounts (github_id
// set) — the avatar next to it carries identity, this marks the auth method.
// "GitHub" is locale-invariant, like the VIP badge.
export function GithubBadge() {
  return (
    <span title="GitHub" className="inline-flex shrink-0 text-muted-foreground">
      <GithubIcon className="size-3.5" aria-hidden="true" />
    </span>
  )
}

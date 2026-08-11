import * as React from 'react'
import { cn } from '@/lib/utils'

const badgeVariants = {
  base: 'inline-flex items-center justify-center rounded-full border border-transparent px-3.5 py-1 text-xs font-semibold w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 gap-1 [&>svg]:pointer-events-none transition-[color,box-shadow] overflow-hidden',
  variant: {
    default: 'bg-primary text-primary-foreground',
    secondary: 'bg-muted text-muted-foreground',
    accent: 'bg-amber text-amber-foreground',
    danger: 'bg-danger text-danger-foreground',
    success: 'bg-success text-success-foreground',
    warning: 'bg-warning text-warning-foreground',
    outline: 'border-border text-foreground',
    ghost: 'text-muted-foreground',
    link: 'text-primary underline-offset-4 hover:underline',
  },
} as const

type BadgeVariant = keyof typeof badgeVariants.variant

function Badge({
  className,
  variant = 'default',
  ...props
}: React.ComponentProps<'span'> & { variant?: BadgeVariant }) {
  return (
    <span
      data-variant={variant}
      className={cn(
        badgeVariants.base,
        badgeVariants.variant[variant],
        className,
      )}
      {...props}
    />
  )
}

export { Badge, type BadgeVariant }

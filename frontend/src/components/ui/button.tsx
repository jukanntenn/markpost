import * as React from 'react'
import { cn } from '@/lib/utils'

const buttonVariants = {
  base: "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-semibold transition-all duration-150 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring",
  variant: {
    default:
      'bg-primary text-primary-foreground hover:bg-primary-hover hover:shadow-primary-glow',
    danger: 'bg-danger text-danger-foreground hover:bg-danger/90',
    outline:
      'border border-border bg-transparent hover:bg-muted hover:text-foreground',
    secondary: 'bg-secondary text-secondary-foreground hover:bg-muted',
    ghost: 'hover:bg-muted hover:text-foreground',
    link: 'text-primary underline-offset-4 hover:underline',
  },
  size: {
    default: 'h-10 px-4 py-2 has-[>svg]:px-3',
    xs: "h-6 gap-1 rounded-md px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
    sm: 'h-8 rounded-md gap-1.5 px-3 has-[>svg]:px-2.5',
    lg: 'h-12 rounded-md px-6 has-[>svg]:px-4',
    icon: 'size-10',
    'icon-xs': "size-6 rounded-md [&_svg:not([class*='size-'])]:size-3",
    'icon-sm': 'size-8',
    'icon-lg': 'size-12',
  },
} as const

type ButtonVariant = keyof typeof buttonVariants.variant
type ButtonSize = keyof typeof buttonVariants.size

const Button = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<'button'> & {
    variant?: ButtonVariant
    size?: ButtonSize
  }
>(({ className, variant = 'default', size = 'default', ...props }, ref) => {
  return (
    <button
      ref={ref}
      data-variant={variant}
      data-size={size}
      className={cn(
        buttonVariants.base,
        buttonVariants.variant[variant],
        buttonVariants.size[size],
        className,
      )}
      {...props}
    />
  )
})

Button.displayName = 'Button'

// buttonClass 供非 <button> 元素（如 <Link>）复用按钮外观。
export function buttonClass(
  variant: ButtonVariant = 'default',
  size: ButtonSize = 'default',
  className?: string,
) {
  return cn(
    buttonVariants.base,
    buttonVariants.variant[variant],
    buttonVariants.size[size],
    className,
  )
}

export { Button, buttonVariants, type ButtonVariant, type ButtonSize }

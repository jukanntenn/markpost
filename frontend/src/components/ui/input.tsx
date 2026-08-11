import * as React from 'react'
import { cn } from '@/lib/utils'

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        ref={ref}
        type={type}
        className={cn(
          'border-input placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground bg-card h-10 w-full min-w-0 rounded-md border px-3 py-2 text-base transition-[color,box-shadow] outline-none disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
          'focus-visible:border-primary focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring',
          'aria-invalid:border-danger',
          className,
        )}
        {...props}
      />
    )
  },
)

Input.displayName = 'Input'

export { Input }

import * as React from 'react'
import { cn } from '@/lib/utils'

const Textarea = React.forwardRef<
  React.ElementRef<'textarea'>,
  React.ComponentProps<'textarea'>
>(({ className, ...props }, ref) => {
  return (
    <textarea
      ref={ref}
      className={cn(
        'border-input placeholder:text-muted-foreground focus-visible:border-primary focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring aria-invalid:border-danger bg-card flex field-sizing-content min-h-16 w-full rounded-md border px-3 py-2 text-base transition-[color,box-shadow] outline-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
        className,
      )}
      {...props}
    />
  )
})

Textarea.displayName = 'Textarea'

export { Textarea }

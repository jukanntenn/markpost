'use client'

import * as React from 'react'
import { Menu as MenuPrimitive } from '@base-ui/react/menu'
import { cn } from '@/lib/utils'

function MenuRoot({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Root>) {
  return <MenuPrimitive.Root {...props} />
}

function MenuTrigger({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Trigger>) {
  return <MenuPrimitive.Trigger {...props} />
}

function MenuPopup({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Popup>) {
  return (
    <MenuPrimitive.Portal>
      <MenuPrimitive.Positioner sideOffset={4}>
        <MenuPrimitive.Popup
          className={cn(
            'z-[100] min-w-[8rem] overflow-x-hidden overflow-y-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-lg outline-none transition-[opacity,transform] duration-150 data-[starting-style]:opacity-0 data-[starting-style]:scale-95 data-[ending-style]:opacity-0 data-[ending-style]:scale-95',
            className,
          )}
          {...props}
        />
      </MenuPrimitive.Positioner>
    </MenuPrimitive.Portal>
  )
}

function MenuGroup({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Group>) {
  return <MenuPrimitive.Group {...props} />
}

function MenuItem({
  className,
  variant = 'default',
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Item> & {
  variant?: 'default' | 'danger'
}) {
  return (
    <MenuPrimitive.Item
      data-variant={variant}
      className={cn(
        "relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        variant === 'danger'
          ? "text-danger data-[highlighted]:bg-danger/10 data-[highlighted]:text-danger [&_svg:not([class*='text-'])]:text-danger"
          : "data-[highlighted]:bg-muted data-[highlighted]:text-foreground [&_svg:not([class*='text-'])]:text-muted-foreground",
        className,
      )}
      {...props}
    />
  )
}

function MenuRadioGroup({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.RadioGroup>) {
  return <MenuPrimitive.RadioGroup {...props} />
}

function MenuRadioItem({
  className,
  children,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.RadioItem>) {
  return (
    <MenuPrimitive.RadioItem
      className={cn(
        "relative flex cursor-default items-center gap-2 rounded-sm py-1.5 pr-2 pl-8 text-sm outline-none select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className,
      )}
      {...props}
    >
      <span className="pointer-events-none absolute left-2 flex size-3.5 items-center justify-center">
        <MenuPrimitive.RadioItemIndicator>
          <span className="size-2 rounded-full bg-current" />
        </MenuPrimitive.RadioItemIndicator>
      </span>
      {children}
    </MenuPrimitive.RadioItem>
  )
}

function MenuLabel({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.GroupLabel>) {
  return (
    <MenuPrimitive.GroupLabel
      className={cn('px-2 py-1.5 text-sm font-semibold', className)}
      {...props}
    />
  )
}

function MenuSeparator({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Separator>) {
  return (
    <MenuPrimitive.Separator
      className={cn('bg-border -mx-1 my-1 h-px', className)}
      {...props}
    />
  )
}

export const Menu = Object.assign(MenuRoot, {
  Root: MenuRoot,
  Trigger: MenuTrigger,
  Popup: MenuPopup,
  Group: MenuGroup,
  Item: MenuItem,
  RadioGroup: MenuRadioGroup,
  RadioItem: MenuRadioItem,
  Label: MenuLabel,
  Separator: MenuSeparator,
})

"use client";

import * as React from "react";
import { Menu as MenuPrimitive } from "@base-ui/react/menu";
import { cn } from "@/lib/utils";

function MenuRoot({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Root>) {
  return <MenuPrimitive.Root data-slot="menu" {...props} />;
}

function MenuTrigger({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Trigger>) {
  return <MenuPrimitive.Trigger data-slot="menu-trigger" {...props} />;
}

function MenuPopup({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Popup>) {
  return (
    <MenuPrimitive.Portal>
      <MenuPrimitive.Positioner sideOffset={4}>
        <MenuPrimitive.Popup
          data-slot="menu-popup"
          className={cn(
            "z-50 min-w-[8rem] overflow-x-hidden overflow-y-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md outline-none transition-[opacity,transform] duration-150 data-[starting-style]:opacity-0 data-[starting-style]:scale-95 data-[ending-style]:opacity-0 data-[ending-style]:scale-95",
            className
          )}
          {...props}
        />
      </MenuPrimitive.Positioner>
    </MenuPrimitive.Portal>
  );
}

function MenuGroup({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Group>) {
  return <MenuPrimitive.Group data-slot="menu-group" {...props} />;
}

function MenuItem({
  className,
  variant = "default",
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Item> & {
  variant?: "default" | "destructive";
}) {
  return (
    <MenuPrimitive.Item
      data-slot="menu-item"
      data-variant={variant}
      className={cn(
        "relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none select-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
        variant === "destructive"
          ? "text-destructive data-[highlighted]:bg-destructive/10 data-[highlighted]:text-destructive [&_svg:not([class*='text-'])]:text-destructive"
          : "data-[highlighted]:bg-muted data-[highlighted]:text-foreground [&_svg:not([class*='text-'])]:text-muted-foreground",
        className
      )}
      {...props}
    />
  );
}

function MenuRadioGroup({
  ...props
}: React.ComponentProps<typeof MenuPrimitive.RadioGroup>) {
  return (
    <MenuPrimitive.RadioGroup data-slot="menu-radio-group" {...props} />
  );
}

function MenuRadioItem({
  className,
  children,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.RadioItem>) {
  return (
    <MenuPrimitive.RadioItem
      data-slot="menu-radio-item"
      className={cn(
        "relative flex cursor-default items-center gap-2 rounded-sm py-1.5 pr-2 pl-8 text-sm outline-none select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
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
  );
}

function MenuLabel({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.GroupLabel>) {
  return (
    <MenuPrimitive.GroupLabel
      data-slot="menu-label"
      className={cn("px-2 py-1.5 text-sm font-medium", className)}
      {...props}
    />
  );
}

function MenuSeparator({
  className,
  ...props
}: React.ComponentProps<typeof MenuPrimitive.Separator>) {
  return (
    <MenuPrimitive.Separator
      data-slot="menu-separator"
      className={cn("bg-border -mx-1 my-1 h-px", className)}
      {...props}
    />
  );
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
});

'use client'

import { Dialog } from '@base-ui/react/dialog'
import { useTranslations } from 'next-intl'
import { XIcon } from 'lucide-react'
import { SidebarNav } from '@/components/layout/SidebarNav'

// A2.5/A2.11 移动端导航：base-ui Dialog 实现的 Sheet，从左滑入。
// portal + scroll lock + focus trap + Esc（base-ui 内建）。路由跳转后由
// AppShell 关闭（onOpenChange 联动）。
export function MobileSidebar({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const t = useTranslations()

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm transition-opacity data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <Dialog.Popup
          aria-label={t('navigation.aria.mobileSidebar')}
          className="fixed inset-y-0 left-0 z-50 flex w-72 max-w-[85vw] flex-col bg-sidebar shadow-modal outline-none transition-transform duration-250 ease-out data-[starting-style]:-translate-x-full data-[ending-style]:-translate-x-full"
        >
          <div className="flex h-(--header-height) shrink-0 items-center justify-between border-b border-sidebar-border px-4">
            <span className="font-display text-lg font-bold text-sidebar-foreground">
              markpost
            </span>
            <Dialog.Close
              aria-label={t('navigation.aria.closeMenu')}
              className="flex size-11 items-center justify-center rounded-md text-sidebar-foreground transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
            >
              <XIcon className="size-5" />
            </Dialog.Close>
          </div>
          <div className="flex-1 overflow-y-auto px-3 py-4">
            <SidebarNav
              onNavigate={() => onOpenChange(false)}
              itemClassName="h-11 [-webkit-tap-highlight-color:transparent]"
            />
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

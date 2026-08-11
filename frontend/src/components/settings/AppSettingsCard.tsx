'use client'

import { useTranslations } from 'next-intl'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { localeNames } from '@/i18n/constants'
import { Select } from '@base-ui/react/select'
import { CheckIcon, ChevronDownIcon } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Label } from '@/components/ui/label'

// F.2 偏好卡片：base-ui Select 替代原生 select（B.2 改造项）。
export function AppSettingsCard() {
  const t = useTranslations('settings')
  const { locale, setLocale, availableLocales } = useLocaleContext()

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('preferences')}</CardTitle>
        <CardDescription>{t('languageDescription')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-4">
          <Label htmlFor="locale-select">{t('language')}</Label>
          <Select.Root
            items={availableLocales.map((l) => ({
              label: localeNames[l],
              value: l,
            }))}
            value={locale}
            onValueChange={(value) => setLocale(value as typeof locale)}
            id="locale-select"
          >
            <Select.Trigger className="flex h-11 min-w-44 items-center justify-between gap-2 rounded-md border border-input bg-card px-3 py-2 text-sm text-foreground transition-[color,box-shadow] select-none focus-visible:border-primary focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring">
              <Select.Value className="data-[placeholder]:text-muted-foreground" />
              <Select.Icon>
                <ChevronDownIcon className="size-4 text-muted-foreground" />
              </Select.Icon>
            </Select.Trigger>
            <Select.Portal>
              <Select.Positioner sideOffset={4}>
                <Select.Popup className="z-[100] min-w-44 overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-lg outline-none">
                  <Select.Arrow />
                  <Select.List>
                    {availableLocales.map((l) => (
                      <Select.Item
                        key={l}
                        value={l}
                        className="flex cursor-default items-center justify-between gap-2 rounded-sm px-2 py-1.5 text-sm outline-none select-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                      >
                        <Select.ItemText>{localeNames[l]}</Select.ItemText>
                        <Select.ItemIndicator className="flex items-center">
                          <CheckIcon className="size-4 text-primary" />
                        </Select.ItemIndicator>
                      </Select.Item>
                    ))}
                  </Select.List>
                </Select.Popup>
              </Select.Positioner>
            </Select.Portal>
          </Select.Root>
        </div>
      </CardContent>
    </Card>
  )
}

/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useMemo, useRef } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const botProtectionSchema = z.object({
  TurnstileCheckEnabled: z.boolean(),
  TurnstileSiteKey: z.string().optional(),
  TurnstileSecretKey: z.string().optional(),
  aliyun_captcha: z.object({
    enabled: z.boolean(),
    region: z.string().optional(),
    prefix: z.string().optional(),
    scene_id: z.string().optional(),
  }),
})

type BotProtectionFormValues = z.infer<typeof botProtectionSchema>

type FlatBotProtectionDefaults = {
  TurnstileCheckEnabled: boolean
  TurnstileSiteKey?: string
  TurnstileSecretKey?: string
  'aliyun_captcha.enabled': boolean
  'aliyun_captcha.region'?: string
  'aliyun_captcha.prefix'?: string
  'aliyun_captcha.scene_id'?: string
}

type BotProtectionSectionProps = {
  defaultValues: FlatBotProtectionDefaults
}

const buildFormDefaults = (
  defaults: FlatBotProtectionDefaults
): BotProtectionFormValues => ({
  TurnstileCheckEnabled: defaults.TurnstileCheckEnabled,
  TurnstileSiteKey: defaults.TurnstileSiteKey ?? '',
  TurnstileSecretKey: defaults.TurnstileSecretKey ?? '',
  aliyun_captcha: {
    enabled: defaults['aliyun_captcha.enabled'],
    region: defaults['aliyun_captcha.region'] || 'cn',
    prefix: defaults['aliyun_captcha.prefix'] ?? '',
    scene_id: defaults['aliyun_captcha.scene_id'] || '1fu9scwz',
  },
})

const normalizeFormValues = (
  values: BotProtectionFormValues
): FlatBotProtectionDefaults => ({
  TurnstileCheckEnabled: values.TurnstileCheckEnabled,
  TurnstileSiteKey: values.TurnstileSiteKey ?? '',
  TurnstileSecretKey: values.TurnstileSecretKey ?? '',
  'aliyun_captcha.enabled': values.aliyun_captcha.enabled,
  'aliyun_captcha.region': values.aliyun_captcha.region || 'cn',
  'aliyun_captcha.prefix': values.aliyun_captcha.prefix ?? '',
  'aliyun_captcha.scene_id': values.aliyun_captcha.scene_id || '1fu9scwz',
})

export function BotProtectionSection({
  defaultValues,
}: BotProtectionSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formDefaults = useMemo(
    () => buildFormDefaults(defaultValues),
    [defaultValues]
  )

  const form = useForm<BotProtectionFormValues>({
    resolver: zodResolver(botProtectionSchema),
    defaultValues: formDefaults,
  })
  const baselineRef = useRef<FlatBotProtectionDefaults>(defaultValues)
  const baselineSerializedRef = useRef(JSON.stringify(defaultValues))

  useEffect(() => {
    const serialized = JSON.stringify(defaultValues)
    if (serialized === baselineSerializedRef.current) return
    baselineRef.current = defaultValues
    baselineSerializedRef.current = serialized
    form.reset(buildFormDefaults(defaultValues))
  }, [defaultValues, form])

  const onSubmit = async (data: BotProtectionFormValues) => {
    const normalized = normalizeFormValues(data)
    const updates = Object.entries(normalized).filter(
      ([key, value]) =>
        value !== baselineRef.current[key as keyof FlatBotProtectionDefaults]
    )

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value: value ?? '' })
    }

    baselineRef.current = normalized
    baselineSerializedRef.current = JSON.stringify(normalized)
    form.reset(buildFormDefaults(normalized))
  }

  return (
    <SettingsSection title={t('Bot Protection')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <div className='text-muted-foreground text-sm lg:col-span-2'>
            {t('Cloudflare Turnstile')}
          </div>
          <FormField
            control={form.control}
            name='TurnstileCheckEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable Turnstile')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Protect login and registration with Cloudflare Turnstile'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='TurnstileSiteKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Site Key')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Your Turnstile site key')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='TurnstileSecretKey'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Secret Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Your Turnstile secret key')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='text-muted-foreground border-t pt-5 text-sm lg:col-span-2'>
            {t('Aliyun ESA AI Captcha')}
          </div>
          <FormField
            control={form.control}
            name='aliyun_captcha.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable Aliyun Captcha')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Require Aliyun ESA AI Captcha before password sign-in'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='aliyun_captcha.prefix'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Identity Prefix')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Aliyun Captcha identity prefix')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('Fill in the Aliyun Captcha identity prefix.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='aliyun_captcha.scene_id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Scene ID')}</FormLabel>
                <FormControl>
                  <Input placeholder='1fu9scwz' autoComplete='off' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='aliyun_captcha.region'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Region')}</FormLabel>
                <FormControl>
                  <Input placeholder='cn' autoComplete='off' {...field} />
                </FormControl>
                <FormDescription>{t('Default is cn.')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

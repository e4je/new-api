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
  aliyunCaptchaEnabled: z.boolean(),
  aliyunCaptchaPrefix: z.string().optional(),
  aliyunCaptchaSceneId: z.string().optional(),
}).superRefine((values, ctx) => {
  if (!values.aliyunCaptchaEnabled) return

  if (!values.aliyunCaptchaPrefix?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['aliyunCaptchaPrefix'],
      message: 'Identity Prefix is required when Aliyun Captcha is enabled.',
    })
  }

  if (!values.aliyunCaptchaSceneId?.trim()) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['aliyunCaptchaSceneId'],
      message: 'Scene ID is required when Aliyun Captcha is enabled.',
    })
  }
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
  aliyunCaptchaEnabled: defaults['aliyun_captcha.enabled'],
  aliyunCaptchaPrefix: defaults['aliyun_captcha.prefix'] ?? '',
  aliyunCaptchaSceneId: defaults['aliyun_captcha.scene_id'] ?? '',
})

const normalizeFormValues = (
  values: BotProtectionFormValues,
  current: FlatBotProtectionDefaults
): FlatBotProtectionDefaults => ({
  TurnstileCheckEnabled: values.TurnstileCheckEnabled,
  TurnstileSiteKey: values.TurnstileSiteKey ?? '',
  TurnstileSecretKey: values.TurnstileSecretKey ?? '',
  'aliyun_captcha.enabled': values.aliyunCaptchaEnabled,
  'aliyun_captcha.region': current['aliyun_captcha.region'] || 'cn',
  'aliyun_captcha.prefix': values.aliyunCaptchaPrefix?.trim() ?? '',
  'aliyun_captcha.scene_id': values.aliyunCaptchaSceneId?.trim() ?? '',
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
    const normalized = normalizeFormValues(data, baselineRef.current)
    const updates = Object.entries(normalized).filter(
      ([key, value]) =>
        value !== baselineRef.current[key as keyof FlatBotProtectionDefaults]
    )
    const orderedUpdates = updates.sort(([leftKey], [rightKey]) => {
      if (leftKey === 'aliyun_captcha.enabled') return 1
      if (rightKey === 'aliyun_captcha.enabled') return -1
      return 0
    })

    for (const [key, value] of orderedUpdates) {
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
            name='aliyunCaptchaEnabled'
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
            name='aliyunCaptchaPrefix'
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
                  {t(
                    'Fill in the identity prefix from Aliyun ESA AI Captcha.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='aliyunCaptchaSceneId'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Scene ID')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Aliyun Captcha scene ID')}
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

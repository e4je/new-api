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
import { useEffect } from 'react'
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
  'aliyun_captcha.enabled': z.boolean(),
  'aliyun_captcha.region': z.string().optional(),
  'aliyun_captcha.prefix': z.string().optional(),
  'aliyun_captcha.scene_id': z.string().optional(),
  'aliyun_captcha.mode': z.string().optional(),
  'aliyun_captcha.script_url': z.string().url().optional().or(z.literal('')),
})

type BotProtectionFormValues = z.infer<typeof botProtectionSchema>

type BotProtectionSectionProps = {
  defaultValues: BotProtectionFormValues
}

export function BotProtectionSection({
  defaultValues,
}: BotProtectionSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<BotProtectionFormValues>({
    resolver: zodResolver(botProtectionSchema),
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const onSubmit = async (data: BotProtectionFormValues) => {
    const updates = Object.entries(data).filter(
      ([key, value]) =>
        value !== defaultValues[key as keyof BotProtectionFormValues]
    )

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value: value ?? '' })
    }
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
            name='aliyun_captcha.region'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Region')}</FormLabel>
                <FormControl>
                  <Input placeholder='cn' autoComplete='off' {...field} />
                </FormControl>
                <FormDescription>{t('Use cn or sgp')}</FormDescription>
                <FormMessage />
              </FormItem>
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
                    placeholder={t('Aliyun Captcha prefix')}
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
            name='aliyun_captcha.mode'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Captcha Mode')}</FormLabel>
                <FormControl>
                  <Input placeholder='popup' autoComplete='off' {...field} />
                </FormControl>
                <FormDescription>{t('Use popup or embed')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='aliyun_captcha.script_url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Script URL')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder='https://o.alicdn.com/captcha-frontend/aliyunCaptcha/AliyunCaptcha.js'
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

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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import type { AliyunCaptchaStatus } from '@/features/auth/types'

declare global {
  interface Window {
    AliyunCaptchaConfig?: {
      region: string
      prefix: string
    }
    initAliyunCaptcha?: (options: {
      SceneId: string
      mode: string
      element: string
      button: string
      success: (captchaVerifyParam: string) => void
      fail?: (result: unknown) => void
      getInstance: (instance: { refresh?: () => void }) => void
      slideStyle?: {
        width: number
        height: number
      }
    }) => void
  }
}

const DEFAULT_SCRIPT_URL =
  'https://o.alicdn.com/captcha-frontend/aliyunCaptcha/AliyunCaptcha.js'

type VerifyCallback = (captchaVerifyParam: string) => void

function loadAliyunCaptchaScript(scriptUrl: string) {
  const existing = document.querySelector<HTMLScriptElement>(
    `script[data-aliyun-captcha="true"]`
  )
  if (existing) {
    return new Promise<void>((resolve, reject) => {
      if (window.initAliyunCaptcha) {
        resolve()
        return
      }
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener('error', () => reject(), { once: true })
    })
  }

  return new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    script.src = scriptUrl
    script.async = true
    script.dataset.aliyunCaptcha = 'true'
    script.addEventListener('load', () => resolve(), { once: true })
    script.addEventListener('error', () => reject(), { once: true })
    document.head.appendChild(script)
  })
}

export function useAliyunCaptcha(config?: AliyunCaptchaStatus) {
  const { t } = useTranslation()
  const [ready, setReady] = useState(false)
  const [initializing, setInitializing] = useState(false)
  const verifyCallbackRef = useRef<VerifyCallback | null>(null)
  const captchaRef = useRef<{ refresh?: () => void } | null>(null)
  const initializedRef = useRef(false)
  const triggerButtonId = 'aliyun-captcha-login-trigger'
  const elementId = 'aliyun-captcha-login-element'

  const enabled = Boolean(
    config?.enabled && config?.prefix && config?.scene_id
  )

  const normalizedConfig = useMemo(
    () => ({
      region: config?.region || 'cn',
      prefix: config?.prefix || '',
      sceneId: config?.scene_id || '',
      mode: config?.mode || 'popup',
      scriptUrl: config?.script_url || DEFAULT_SCRIPT_URL,
    }),
    [
      config?.mode,
      config?.prefix,
      config?.region,
      config?.scene_id,
      config?.script_url,
    ]
  )

  useEffect(() => {
    if (!enabled || initializedRef.current) return

    let cancelled = false
    setInitializing(true)
    window.AliyunCaptchaConfig = {
      region: normalizedConfig.region,
      prefix: normalizedConfig.prefix,
    }

    loadAliyunCaptchaScript(normalizedConfig.scriptUrl)
      .then(() => {
        if (cancelled) return
        if (!window.initAliyunCaptcha) {
          throw new Error('initAliyunCaptcha is not available')
        }

        window.initAliyunCaptcha({
          SceneId: normalizedConfig.sceneId,
          mode: normalizedConfig.mode,
          element: `#${elementId}`,
          button: `#${triggerButtonId}`,
          success: (captchaVerifyParam) => {
            verifyCallbackRef.current?.(captchaVerifyParam)
            captchaRef.current?.refresh?.()
          },
          fail: () => {
            toast.error(t('Captcha verification failed'))
          },
          getInstance: (instance) => {
            captchaRef.current = instance
          },
          slideStyle: {
            width: 320,
            height: 40,
          },
        })

        initializedRef.current = true
        setReady(true)
      })
      .catch(() => {
        if (!cancelled) {
          toast.error(t('Failed to load Aliyun Captcha'))
        }
      })
      .finally(() => {
        if (!cancelled) {
          setInitializing(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [
    elementId,
    enabled,
    normalizedConfig.mode,
    normalizedConfig.prefix,
    normalizedConfig.region,
    normalizedConfig.sceneId,
    normalizedConfig.scriptUrl,
    t,
    triggerButtonId,
  ])

  const trigger = useCallback(
    (callback: VerifyCallback) => {
      if (!enabled) return false
      if (!ready) {
        toast.error(
          initializing
            ? t('Captcha is still loading')
            : t('Captcha is not ready')
        )
        return true
      }

      verifyCallbackRef.current = callback
      document.getElementById(triggerButtonId)?.click()
      return true
    },
    [enabled, initializing, ready, t, triggerButtonId]
  )

  return {
    enabled,
    ready,
    initializing,
    elementId,
    triggerButtonId,
    trigger,
  }
}

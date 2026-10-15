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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { UserAuthForm } from '../user-auth-form'

function renderLogin(captcha = false) {
  const client = new QueryClient()
  client.setQueryData(STATUS_QUERY_KEY, {
    password_login_enabled: true,
    user_agreement_enabled: true,
    aliyun_captcha: {
      enabled: captcha,
      prefix: 'test',
      scene_id: 'test-scene',
    },
  })
  const router = createRouter({
    routeTree: createRootRoute({ component: UserAuthForm }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return client
}

afterEach(() => {
  cleanup()
  document.querySelector('script[data-aliyun-captcha="true"]')?.remove()
  delete window.AliyunCaptchaConfig
  vi.restoreAllMocks()
})

it('shows the legal warning without sending credentials when consent is unchecked', async () => {
  const error = vi.spyOn(toast, 'error')
  const post = vi.spyOn(api, 'post')
  const client = renderLogin()
  const user = userEvent.setup()
  const button = await screen.findByRole('button', { name: 'Sign in' })
  expect(button).not.toBeDisabled()
  await user.type(screen.getByLabelText('Username or Email'), 'test-user')
  await user.type(screen.getByLabelText('Password'), 'test-password')
  await user.click(button)
  await waitFor(() =>
    expect(error).toHaveBeenCalledWith('Please agree to the legal terms first')
  )
  expect(post).not.toHaveBeenCalled()
  client.clear()
})

it('keeps login disabled while the configured Aliyun captcha is loading', async () => {
  const client = renderLogin(true)
  const button = await screen.findByRole('button', { name: 'Sign in' })
  await waitFor(() => expect(button).toBeDisabled())
  expect(
    document.querySelector('script[data-aliyun-captcha="true"]')
  ).not.toBeNull()
  client.clear()
})

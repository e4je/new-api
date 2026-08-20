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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { updateChannelUsedQuota } from '../../../api'
import { UsedQuotaDialog } from '../used-quota-dialog'

vi.mock('../../../api', () => ({
  updateChannelUsedQuota: vi.fn(),
}))

describe('UsedQuotaDialog', () => {
  beforeEach(() => {
    vi.mocked(updateChannelUsedQuota).mockReset()
  })

  test('allows an administrator to reset accumulated channel usage to zero', async () => {
    const user = userEvent.setup()
    const onOpenChange = vi.fn()
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    vi.mocked(updateChannelUsedQuota).mockResolvedValue({ success: true })

    render(
      <QueryClientProvider client={queryClient}>
        <UsedQuotaDialog
          open
          onOpenChange={onOpenChange}
          channelId={42}
          channelName='monthly-provider'
          currentUsedQuota={1_000_000}
        />
      </QueryClientProvider>
    )

    const input = screen.getByRole('spinbutton', { name: /Used/ })
    await user.clear(input)
    await user.type(input, '0')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(updateChannelUsedQuota).toHaveBeenCalledWith(42, 0)
    })
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})

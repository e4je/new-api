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
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { getCurrencyLabel } from '@/lib/currency'
import {
  formatQuota,
  getEditableQuotaStep,
  parseQuotaFromDollars,
  quotaUnitsToEditableAmount,
} from '@/lib/format'

import { updateChannelUsedQuota } from '../../api'
import { channelsQueryKeys } from '../../lib'

interface UsedQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelId: number
  channelName: string
  currentUsedQuota: number
}

export function UsedQuotaDialog(props: UsedQuotaDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [amount, setAmount] = useState(() =>
    quotaUnitsToEditableAmount(props.currentUsedQuota).toString()
  )
  const [loading, setLoading] = useState(false)

  const parsedAmount = amount.trim() === '' ? Number.NaN : Number(amount)
  const nextUsedQuota = parseQuotaFromDollars(parsedAmount)
  const isNegative = Number.isFinite(parsedAmount) && parsedAmount < 0
  const isValid =
    Number.isFinite(parsedAmount) &&
    parsedAmount >= 0 &&
    Number.isSafeInteger(nextUsedQuota)
  let error: string | undefined
  if (isNegative) {
    error = t('Value must be at least 0')
  } else if (amount.trim() !== '' && !isValid) {
    error = t('Please enter a valid number')
  }

  const handleClose = () => {
    if (!loading) props.onOpenChange(false)
  }

  const handleSave = async () => {
    if (!isValid) return

    setLoading(true)
    try {
      const result = await updateChannelUsedQuota(
        props.channelId,
        nextUsedQuota
      )
      if (!result.success) {
        toast.error(result.message || t('Failed to update channel'))
        return
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() }),
        queryClient.invalidateQueries({
          queryKey: channelsQueryKeys.detail(props.channelId),
        }),
      ])
      toast.success(t('Channel updated successfully'))
      props.onOpenChange(false)
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to update channel')
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleClose}
      title={t('Edit used quota')}
      description={t(
        'Change the accumulated usage for "{{name}}". Usage logs are not affected.',
        { name: props.channelName }
      )}
      contentHeight='auto'
      footer={
        <>
          <Button variant='outline' onClick={handleClose} disabled={loading}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleSave} disabled={!isValid || loading}>
            {loading && <Spinner data-icon='inline-start' />}
            {t('Save')}
          </Button>
        </>
      }
    >
      <FieldGroup>
        <Field data-invalid={Boolean(error)}>
          <FieldLabel htmlFor='channel-used-quota'>
            {t('Used')} ({getCurrencyLabel()})
          </FieldLabel>
          <Input
            id='channel-used-quota'
            type='number'
            min={0}
            step={getEditableQuotaStep()}
            value={amount}
            aria-invalid={Boolean(error)}
            onChange={(event) => setAmount(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') handleSave()
            }}
          />
          <FieldDescription>
            {t('Current quota')}: {formatQuota(props.currentUsedQuota)}
          </FieldDescription>
          <FieldError>{error}</FieldError>
        </Field>
      </FieldGroup>
    </Dialog>
  )
}

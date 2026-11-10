import { useEffect, useState } from 'react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import type { ContactConflict } from '@/types/api/registration'
import { useI18n } from '@/i18n'

export function ContactConflictDialog() {
  const { t } = useI18n()
  const [conflict, setConflict] = useState<ContactConflict | null>(null)
  useEffect(() => {
    const receive = (event: Event) => setConflict((event as CustomEvent<ContactConflict>).detail)
    window.addEventListener('auth:contact-conflict', receive)
    return () => window.removeEventListener('auth:contact-conflict', receive)
  }, [])
  return (
    <Dialog
      isOpen={!!conflict}
      onOpenChange={(open) => {
        if (!open) setConflict(null)
      }}
    >
      <DialogHeader>
        <DialogTitle>{t('registration.conflictTitle')}</DialogTitle>
        <DialogDescription>
          {conflict
            ? t(
                conflict.contact === 'qq'
                  ? 'registration.qqConflict'
                  : 'registration.discordConflict',
                { name: conflict.primary_character_name }
              )
            : ''}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter showCloseButton />
    </Dialog>
  )
}

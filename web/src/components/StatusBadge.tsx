import type { MigrationStatus } from '../lib/api'

type StatusBadgeProps = {
  status: MigrationStatus
  tone?: 'danger'
}

const statusStyles: Record<MigrationStatus, string> = {
  applied: 'border-secondary/30 bg-secondary-container/10 text-secondary',
  pending: 'border-outline-variant bg-surface-container-low text-on-surface-variant',
  'rolled-back': 'border-tertiary/30 bg-tertiary-container/20 text-tertiary',
  failed: 'border-error/50 bg-error-container/10 text-error',
}

export function StatusBadge({ status, tone }: StatusBadgeProps) {
  const style = tone === 'danger' ? 'border-error/50 bg-error-container/10 text-error' : statusStyles[status]
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 font-label-caps text-label-caps uppercase ${style}`}>
      {tone === 'danger' ? <span className="material-symbols-outlined text-[14px]">warning</span> : <span className="h-1.5 w-1.5 rounded-full bg-current" />}
      {status}
    </span>
  )
}

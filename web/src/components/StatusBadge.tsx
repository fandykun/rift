import type { MigrationStatus } from '../lib/api'

type StatusBadgeProps = {
  status: MigrationStatus
}

const statusStyles: Record<MigrationStatus, string> = {
  applied: 'border-secondary/30 bg-secondary-container/20 text-secondary',
  pending: 'border-outline-variant bg-surface-container-low text-on-surface-variant',
  'rolled-back': 'border-tertiary/30 bg-tertiary-container/20 text-tertiary',
  failed: 'border-error/50 bg-error-container/10 text-error',
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded border px-unit-2 py-unit-1 font-label-caps text-label-caps uppercase ${statusStyles[status]}`}
    >
      {status}
    </span>
  )
}

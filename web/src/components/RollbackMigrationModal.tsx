import type { Migration } from '../lib/api'

type RollbackMigrationModalProps = {
  migration: Migration
  error?: string
  isPending: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function RollbackMigrationModal({
  migration,
  error,
  isPending,
  onCancel,
  onConfirm,
}: RollbackMigrationModalProps) {
  return (
    <div
      aria-labelledby="rollback-modal-title"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 p-unit-4 backdrop-blur-sm"
      role="dialog"
    >
      <section className="w-full max-w-lg rounded-xl border border-error/50 bg-surface-container-highest p-unit-6">
        <div className="flex items-start gap-unit-4">
          <span className="material-symbols-outlined text-error" aria-hidden="true">
            warning
          </span>
          <div>
            <h2 id="rollback-modal-title" className="font-headline-md text-headline-md text-on-surface">
              Rollback latest migration?
            </h2>
            <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">
              Rift will execute this migration&apos;s down SQL against the connected database. Review the file before continuing.
            </p>
          </div>
        </div>

        <div className="mt-unit-4 rounded border border-outline-variant bg-surface-container-lowest p-unit-4">
          <p className="font-label-caps text-label-caps uppercase text-on-surface-variant">Migration</p>
          <p className="mt-unit-2 break-all font-code-sm text-code-sm text-primary">{migration.filename}</p>
          <p className="mt-unit-1 font-code-sm text-code-sm text-on-surface-variant">Version {migration.version}</p>
        </div>

        {error ? (
          <p className="mt-unit-4 rounded border border-error/50 bg-error-container/10 p-3 font-body-md text-body-md text-error" role="alert">
            {error}
          </p>
        ) : null}

        <div className="mt-unit-6 flex justify-end gap-unit-2">
          <button
            className="rounded border border-outline-variant px-4 py-2 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant disabled:cursor-not-allowed disabled:opacity-50"
            disabled={isPending}
            type="button"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button
            className="rounded border border-error/50 bg-error/10 px-4 py-2 font-label-caps text-label-caps font-bold uppercase text-error hover:bg-error/20 disabled:cursor-not-allowed disabled:opacity-50"
            disabled={isPending}
            type="button"
            onClick={onConfirm}
          >
            {isPending ? 'Rolling back…' : 'Confirm rollback'}
          </button>
        </div>
      </section>
    </div>
  )
}

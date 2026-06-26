type QuickActionsCardProps = {
  connectMessage?: string
  errorMessage?: string
  isConnecting?: boolean
  isSyncing?: boolean
  syncMessage?: string
  onConnect: () => void
  onSync: () => void
}

export function QuickActionsCard({
  connectMessage,
  errorMessage,
  isConnecting = false,
  isSyncing = false,
  syncMessage,
  onConnect,
  onSync,
}: QuickActionsCardProps) {
  const actionInProgress = isConnecting || isSyncing

  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <h3 className="font-headline-sm text-headline-sm text-on-surface">Quick Actions</h3>
      <div className="mt-unit-4 space-y-unit-2">
        <button
          className="flex w-full items-center justify-center gap-unit-2 rounded bg-primary px-3 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={actionInProgress}
          type="button"
          onClick={onConnect}
        >
          <span className="material-symbols-outlined text-[18px]">database</span>
          {isConnecting ? 'Checking DB…' : 'Connect to DB'}
        </button>
        <button
          className="flex w-full items-center justify-center gap-unit-2 rounded border border-outline-variant px-3 py-2 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant disabled:cursor-not-allowed disabled:opacity-50"
          disabled={actionInProgress}
          type="button"
          onClick={onSync}
        >
          <span className="material-symbols-outlined text-[18px]">sync</span>
          {isSyncing ? 'Syncing…' : 'Sync Local Files'}
        </button>
      </div>

      {connectMessage ? (
        <p className="mt-unit-3 rounded border border-secondary/30 bg-secondary/10 p-unit-2 font-body-md text-body-md text-secondary">
          {connectMessage}
        </p>
      ) : null}
      {syncMessage ? (
        <p className="mt-unit-3 rounded border border-primary/30 bg-primary-container/10 p-unit-2 font-body-md text-body-md text-primary">
          {syncMessage}
        </p>
      ) : null}
      {errorMessage ? (
        <p className="mt-unit-3 rounded border border-error/40 bg-error-container/10 p-unit-2 font-body-md text-body-md text-error">
          {errorMessage}
        </p>
      ) : null}
    </section>
  )
}

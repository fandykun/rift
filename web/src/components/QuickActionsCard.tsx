export function QuickActionsCard() {
  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <h3 className="font-headline-sm text-headline-sm text-on-surface">Quick Actions</h3>
      <div className="mt-unit-4 space-y-unit-2">
        <button className="flex w-full items-center justify-center gap-unit-2 rounded bg-primary px-3 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary hover:bg-primary/90" type="button">
          <span className="material-symbols-outlined text-[18px]">database</span>
          Connect to DB
        </button>
        <button className="flex w-full items-center justify-center gap-unit-2 rounded border border-outline-variant px-3 py-2 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant" type="button">
          <span className="material-symbols-outlined text-[18px]">sync</span>
          Sync Local Files
        </button>
      </div>
    </section>
  )
}

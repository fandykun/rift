function App() {
  return (
    <main className="min-h-screen bg-background text-on-surface">
      <aside className="fixed inset-y-0 left-0 hidden w-sidebar-width border-r border-outline-variant bg-surface-container p-unit-4 lg:flex lg:flex-col">
        <div className="mb-unit-8 flex items-center gap-unit-2">
          <div className="flex h-8 w-8 items-center justify-center rounded bg-primary-container text-on-primary-container">
            <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>
              database
            </span>
          </div>
          <div>
            <h1 className="font-display text-headline-sm font-bold text-primary">Rift DB</h1>
            <p className="font-label-caps text-label-caps uppercase text-on-surface-variant">
              PostgreSQL Instance
            </p>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-unit-1 font-body-md text-body-md">
          <a className="flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 text-primary transition-colors border-r-2 border-primary bg-primary-container/10">
            <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>
              dashboard
            </span>
            <span className="font-medium">Dashboard</span>
          </a>
          <a className="flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 text-on-surface-variant transition-colors hover:bg-surface-variant">
            <span className="material-symbols-outlined">database_off</span>
            <span>Migrations</span>
          </a>
          <a className="flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 text-on-surface-variant transition-colors hover:bg-surface-variant">
            <span className="material-symbols-outlined">difference</span>
            <span>Schema Diff</span>
          </a>
          <a className="flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 text-on-surface-variant transition-colors hover:bg-surface-variant">
            <span className="material-symbols-outlined">group</span>
            <span>Team</span>
          </a>
          <a className="flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 text-on-surface-variant transition-colors hover:bg-surface-variant">
            <span className="material-symbols-outlined">settings</span>
            <span>Settings</span>
          </a>
        </nav>

        <button className="mb-unit-4 flex w-full items-center justify-center gap-unit-2 rounded bg-primary py-2 font-body-md text-body-md font-medium text-on-primary transition-colors hover:bg-primary/90 active:scale-95">
          <span className="material-symbols-outlined text-[18px]">add</span>
          New Migration
        </button>
      </aside>

      <section className="lg:pl-sidebar-width">
        <header className="sticky top-0 flex h-14 items-center justify-between border-b border-outline-variant bg-background px-gutter">
          <div className="flex gap-unit-4 font-label-caps text-label-caps uppercase text-on-surface-variant">
            <a className="hover:text-on-surface">Docs</a>
            <a className="hover:text-on-surface">API</a>
            <a className="hover:text-on-surface">Changelog</a>
          </div>
          <div className="flex items-center gap-unit-2">
            <button className="rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant">
              Preview Changes
            </button>
            <button className="rounded bg-primary px-3 py-1.5 font-label-caps text-label-caps font-bold uppercase text-on-primary transition-colors hover:bg-primary/90">
              Apply Migrations
            </button>
          </div>
        </header>

        <div className="mx-auto max-w-container-max p-unit-8">
          <div className="mb-unit-8">
            <h2 className="font-display text-display text-on-background">Migration Dashboard</h2>
            <p className="font-body-md text-body-md text-on-surface-variant">
              Manage and track database schema evolution.
            </p>
          </div>

          <div className="grid gap-unit-4 md:grid-cols-3">
            {[
              ['Total Migrations', '0', 'database'],
              ['Applied', '0', 'check_circle'],
              ['Pending', '0', 'pending_actions'],
            ].map(([label, value, icon]) => (
              <section
                key={label}
                className="group relative flex flex-col gap-unit-2 overflow-hidden rounded border border-outline-variant bg-surface-container p-unit-4"
              >
                <span className="font-label-caps text-label-caps uppercase text-on-surface-variant">
                  {label}
                </span>
                <strong className="font-headline-md text-headline-md text-on-background">{value}</strong>
                <span className="material-symbols-outlined absolute right-0 top-0 p-unit-4 text-4xl opacity-20 transition-opacity group-hover:opacity-40">
                  {icon}
                </span>
              </section>
            ))}
          </div>
        </div>
      </section>
    </main>
  )
}

export default App

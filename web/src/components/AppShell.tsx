import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { createMigration, fetchMigrations } from '../lib/api'
import type { Migration } from '../lib/api'
import { useAppStore } from '../stores/appStore'

type NavItem = {
  to: string
  icon: string
  label: string
  isActive: (pathname: string) => boolean
}

function buildNavItems(pathname: string, migrations: Migration[] = []): NavItem[] {
  const diffPath = buildSchemaDiffPath(pathname, migrations)
  return [
    { to: '/', icon: 'dashboard', label: 'Dashboard', isActive: (path) => path === '/' },
    {
      to: '/migrations',
      icon: 'database_off',
      label: 'Migrations',
      isActive: (path) => path === '/migrations' || /^\/migrations\/[^/]+$/.test(path),
    },
    { to: diffPath, icon: 'difference', label: 'Schema Diff', isActive: (path) => /^\/migrations\/[^/]+\/diff$/.test(path) },
    { to: '/team', icon: 'group', label: 'Team', isActive: (path) => path === '/team' },
    { to: '/settings', icon: 'settings', label: 'Settings', isActive: (path) => path === '/settings' },
  ]
}

function buildSchemaDiffPath(pathname: string, migrations: Migration[]): string {
  if (/^\/migrations\/[^/]+\/diff$/.test(pathname)) {
    return pathname
  }

  const currentMigrationMatch = pathname.match(/^\/migrations\/([^/]+)$/)
  if (currentMigrationMatch?.[1]) {
    return `/migrations/${currentMigrationMatch[1]}/diff`
  }

  const preferredMigration = migrations.find((migration) => migration.status === 'pending') ?? migrations[0]
  if (preferredMigration) {
    return `/migrations/${preferredMigration.version}/diff`
  }

  return '/migrations'
}

export function AppShell() {
  const environmentName = useAppStore((state) => state.environmentName)
  const token = useAppStore((state) => state.apiToken)
  const theme = useAppStore((state) => state.theme)
  const toggleTheme = useAppStore((state) => state.toggleTheme)
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const migrationsQuery = useQuery({
    queryKey: ['migrations', token],
    queryFn: () => fetchMigrations({ token }),
    enabled: Boolean(token),
  })
  const createMigrationMutation = useMutation({
    mutationFn: (name: string) => createMigration({ name }, { token }),
    onSuccess: async (migration) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['migrations'] }),
        queryClient.invalidateQueries({ queryKey: ['status'] }),
        queryClient.invalidateQueries({ queryKey: ['lint'] }),
      ])
      setCreateModalOpen(false)
      void navigate(`/migrations/${migration.version}`)
    },
  })
  const migrations = migrationsQuery.data ?? []
  const navItems = buildNavItems(pathname, migrations)
  const schemaDiffPath = buildSchemaDiffPath(pathname, migrations)
  const canPreviewChanges = Boolean(token && schemaDiffPath !== '/migrations')

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

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
              {environmentName}
            </p>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-unit-1 font-body-md text-body-md">
          {navItems.map((item) => {
            const isActive = item.isActive(pathname)
            return (
              <Link
                key={`${item.to}-${item.label}`}
                to={item.to}
                className={[
                  'flex cursor-pointer items-center gap-unit-2 rounded px-unit-2 py-2 transition-colors',
                  isActive
                    ? 'border-r-2 border-primary bg-primary-container/10 text-primary'
                    : 'text-on-surface-variant hover:bg-surface-variant hover:text-on-surface',
                ].join(' ')}
              >
                <span className="material-symbols-outlined">{item.icon}</span>
                <span> {item.label}</span>
              </Link>
            )
          })}
        </nav>

        <button
          className="mb-unit-4 flex w-full items-center justify-center gap-unit-2 rounded bg-primary py-2 font-body-md text-body-md font-medium text-on-primary transition-colors hover:bg-primary/90 active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={!token}
          type="button"
          onClick={() => setCreateModalOpen(true)}
        >
          <span className="material-symbols-outlined text-[18px]">add</span>
          New Migration
        </button>
      </aside>

      <section className="lg:pl-sidebar-width">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-outline-variant bg-background px-gutter">
          <div className="flex gap-unit-4 font-label-caps text-label-caps uppercase text-on-surface-variant">
            <a className="hover:text-on-surface" href="/README.md">
              Docs
            </a>
            <a className="hover:text-on-surface" href="/api/v1/status">
              API
            </a>
            <span>Rift</span>
          </div>
          <div className="flex items-center gap-unit-2">
            <button
              aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
              className="flex items-center gap-unit-2 rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant"
              type="button"
              onClick={toggleTheme}
            >
              <span className="material-symbols-outlined text-[18px]">{theme === 'dark' ? 'light_mode' : 'dark_mode'}</span>
              {theme === 'dark' ? 'Light' : 'Dark'} Mode
            </button>
            <button
              className="rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant disabled:cursor-not-allowed disabled:opacity-40"
              disabled={!canPreviewChanges}
              type="button"
              onClick={() => void navigate(schemaDiffPath)}
            >
              Preview Changes
            </button>
            <button
              className="rounded bg-primary px-3 py-1.5 font-label-caps text-label-caps font-bold uppercase text-on-primary transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40"
              disabled={!canPreviewChanges}
              type="button"
              onClick={() => void navigate(`${schemaDiffPath}?apply=1`)}
            >
              Apply Migrations
            </button>
          </div>
        </header>

        <div className="mx-auto max-w-container-max p-unit-8">
          <Outlet />
        </div>
      </section>

      {createModalOpen ? (
        <CreateMigrationModal
          error={createMigrationMutation.error instanceof Error ? createMigrationMutation.error.message : undefined}
          isCreating={createMigrationMutation.isPending}
          onClose={() => setCreateModalOpen(false)}
          onCreate={(name) => createMigrationMutation.mutate(name)}
        />
      ) : null}
    </main>
  )
}


type CreateMigrationModalProps = {
  error?: string
  isCreating: boolean
  onClose: () => void
  onCreate: (name: string) => void
}

function CreateMigrationModal({ error, isCreating, onClose, onCreate }: CreateMigrationModalProps) {
  const [name, setName] = useState('')
  const normalizedName = normalizeMigrationName(name)
  const canCreate = normalizedName.length > 0 && !isCreating

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/70 p-unit-8 backdrop-blur-sm">
      <section className="w-full max-w-lg rounded-xl border border-outline-variant bg-surface-container-highest p-unit-6 shadow-2xl">
        <div className="flex items-start justify-between gap-unit-4">
          <div>
            <h2 className="font-headline-md text-headline-md text-on-surface">New migration</h2>
            <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">
              Create a timestamped up/down SQL pair in the configured migrations directory.
            </p>
          </div>
          <button aria-label="Close new migration dialog" className="text-on-surface-variant hover:text-on-surface" type="button" onClick={onClose}>
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>

        <form
          className="mt-unit-5 space-y-unit-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (canCreate) {
              onCreate(name)
            }
          }}
        >
          <label className="block font-body-md text-body-md text-on-surface">
            Migration name
            <input
              autoFocus
              className="mt-unit-2 w-full rounded border border-outline-variant bg-surface-container-lowest px-unit-3 py-2 font-body-md text-body-md text-on-surface outline-none transition-colors placeholder:text-on-surface-variant focus:border-primary"
              placeholder="add billing events"
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </label>

          <div className="rounded border border-outline-variant bg-surface-container p-unit-3 font-code-sm text-code-sm text-on-surface-variant">
            <span className="text-on-surface">File preview:</span>{' '}
            {normalizedName ? `YYYYMMDD_HHMMSS_${normalizedName}.up.sql` : 'Enter a name to preview the filename'}
          </div>

          {error ? <p className="rounded border border-error/40 bg-error-container/10 p-unit-3 font-body-md text-body-md text-error">{error}</p> : null}

          <div className="flex justify-end gap-unit-2">
            <button className="rounded border border-outline-variant px-4 py-2 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant" type="button" onClick={onClose}>
              Cancel
            </button>
            <button
              className="rounded bg-primary px-4 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary disabled:cursor-not-allowed disabled:opacity-40"
              disabled={!canCreate}
              type="submit"
            >
              {isCreating ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </section>
    </div>
  )
}

function normalizeMigrationName(rawName: string): string {
  return rawName
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQueries } from '@tanstack/react-query'
import { DataTable } from '../components/DataTable'
import { ErrorState } from '../components/ErrorState'
import { LinterAlertsCard } from '../components/LinterAlertsCard'
import { LoadingSkeleton } from '../components/LoadingSkeleton'
import { QuickActionsCard } from '../components/QuickActionsCard'
import { StatCard } from '../components/StatCard'
import { StatusBadge } from '../components/StatusBadge'
import { fetchLint, fetchMigrations, fetchStatus } from '../lib/api'
import type { LintResponse, Migration, StatusResponse } from '../lib/api'
import { useAppStore } from '../stores/appStore'

const columns = ['Status', 'ID', 'Name', 'Author', 'Applied Date', 'Actions']

export function MigrationsPage() {
  const token = useAppStore((state) => state.apiToken)
  const setApiToken = useAppStore((state) => state.setApiToken)
  const setEnvironmentName = useAppStore((state) => state.setEnvironmentName)
  const [draftToken, setDraftToken] = useState(token)
  const [searchTerm, setSearchTerm] = useState('')

  const [statusQuery, migrationsQuery, lintQuery] = useQueries({
    queries: [
      {
        queryKey: ['status', token],
        queryFn: () => fetchStatus({ token }),
        enabled: token.length > 0,
      },
      {
        queryKey: ['migrations', token],
        queryFn: () => fetchMigrations({ token }),
        enabled: token.length > 0,
      },
      {
        queryKey: ['lint', token],
        queryFn: () => fetchLint({ token }),
        enabled: token.length > 0,
      },
    ],
  })

  const status = statusQuery.data as StatusResponse | undefined
  const rawMigrations = migrationsQuery.data as Migration[] | undefined
  const migrations = useMemo(() => rawMigrations ?? [], [rawMigrations])
  const lint = (lintQuery.data as LintResponse | undefined) ?? { error_count: 0, warning_count: 0, results: [] }

  useEffect(() => {
    if (status?.environment) {
      setEnvironmentName(status.environment)
    }
  }, [setEnvironmentName, status?.environment])

  const filteredMigrations = useMemo(() => {
    const normalized = searchTerm.trim().toLowerCase()
    if (!normalized) {
      return migrations
    }
    return migrations.filter((migration) =>
      [migration.version, migration.filename, migration.applied_by ?? ''].some((value) =>
        value.toLowerCase().includes(normalized),
      ),
    )
  }, [migrations, searchTerm])

  if (!token) {
    return (
      <TokenPrompt
        draftToken={draftToken}
        onDraftTokenChange={setDraftToken}
        onSave={() => setApiToken(draftToken.trim())}
      />
    )
  }

  if (statusQuery.isLoading || migrationsQuery.isLoading || lintQuery.isLoading) {
    return <LoadingSkeleton />
  }

  const firstError = statusQuery.error ?? migrationsQuery.error ?? lintQuery.error
  if (firstError instanceof Error) {
    return <ErrorState message={firstError.message} onRetry={() => void Promise.all([statusQuery.refetch(), migrationsQuery.refetch(), lintQuery.refetch()])} />
  }

  return (
    <div>
      <div className="mb-unit-8">
        <h2 className="font-display text-display text-on-background">Migration Dashboard</h2>
        <p className="font-body-md text-body-md text-on-surface-variant">
          Manage and track database schema evolution.
        </p>
      </div>

      <div className="grid gap-unit-4 md:grid-cols-3">
        <StatCard icon="database" label="Total Migrations" value={status?.counts.total ?? migrations.length} />
        <StatCard icon="check_circle" label="Applied" value={status?.counts.applied ?? 0} />
        <StatCard icon="pending_actions" label="Pending" value={status?.counts.pending ?? 0} />
      </div>

      <div className="mt-unit-6 grid gap-unit-4 xl:grid-cols-12">
        <section className="xl:col-span-8">
          <div className="mb-unit-4 flex items-center justify-between gap-unit-4">
            <h3 className="font-headline-sm text-headline-sm text-on-surface">Recent Activity</h3>
            <label className="relative w-72 max-w-full">
              <span className="material-symbols-outlined absolute left-2 top-1/2 -translate-y-1/2 text-[18px] text-on-surface-variant">
                search
              </span>
              <input
                className="w-full rounded border border-outline-variant bg-surface-container py-1.5 pl-9 pr-3 font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant focus:border-primary focus:ring-primary"
                placeholder="Search migrations"
                type="search"
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
              />
            </label>
          </div>

          {filteredMigrations.length > 0 ? (
            <DataTable columns={columns}>
              {filteredMigrations.map((migration) => (
                <MigrationRow key={`${migration.version}-${migration.status}`} migration={migration} />
              ))}
            </DataTable>
          ) : (
            <EmptyMigrations />
          )}
        </section>

        <aside className="space-y-unit-4 xl:col-span-4">
          <QuickActionsCard />
          <LinterAlertsCard results={lint.results} />
        </aside>
      </div>
    </div>
  )
}

type TokenPromptProps = {
  draftToken: string
  onDraftTokenChange: (token: string) => void
  onSave: () => void
}

function TokenPrompt({ draftToken, onDraftTokenChange, onSave }: TokenPromptProps) {
  return (
    <section className="mx-auto max-w-xl rounded border border-outline-variant bg-surface-container p-unit-6">
      <h2 className="font-display text-headline-md text-on-background">Connect to Rift API</h2>
      <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">
        Enter the bearer token configured in <code className="rounded bg-surface-variant px-1 text-primary">rift.yaml</code>.
      </p>
      <div className="mt-unit-4 flex gap-unit-2">
        <input
          className="min-w-0 flex-1 rounded border border-outline-variant bg-surface-container-low px-3 py-2 font-code-sm text-code-sm text-on-surface focus:border-primary focus:ring-primary"
          placeholder="API token"
          type="password"
          value={draftToken}
          onChange={(event) => onDraftTokenChange(event.target.value)}
        />
        <button className="rounded bg-primary px-4 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary" type="button" onClick={onSave}>
          Save
        </button>
      </div>
    </section>
  )
}

function MigrationRow({ migration }: { migration: Migration }) {
  const name = migration.filename.replace(`${migration.version}_`, '').replace(/\.up\.sql$/, '')

  return (
    <tr className="bg-surface-container transition-colors hover:bg-surface-container-low">
      <td className="px-unit-4 py-3">
        <StatusBadge status={migration.status} tone={migration.has_lint ? 'danger' : undefined} />
      </td>
      <td className="px-unit-4 py-3 font-code-sm text-code-sm text-primary">{migration.version}</td>
      <td className="px-unit-4 py-3 text-on-surface">{name}</td>
      <td className="px-unit-4 py-3 text-on-surface-variant">{migration.applied_by ?? 'local'}</td>
      <td className="px-unit-4 py-3 font-code-sm text-code-sm text-on-surface-variant">
        {migration.applied_at ? new Date(migration.applied_at).toLocaleString() : '—'}
      </td>
      <td className="px-unit-4 py-3">
        <div className="flex gap-unit-2">
          <Link className="font-label-caps text-label-caps uppercase text-primary hover:underline" to={`/migrations/${migration.version}`}>
            View
          </Link>
          <Link className="font-label-caps text-label-caps uppercase text-tertiary hover:underline" to={`/migrations/${migration.version}/diff`}>
            Diff
          </Link>
        </div>
      </td>
    </tr>
  )
}

function EmptyMigrations() {
  return (
    <div className="flex min-h-80 flex-col items-center justify-center rounded border border-outline-variant bg-surface-container p-unit-8 text-center">
      <span className="material-symbols-outlined text-5xl text-on-surface-variant/30">database</span>
      <h3 className="mt-unit-4 font-headline-sm text-headline-sm text-on-surface">No migrations yet</h3>
      <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">Run `rift new &lt;name&gt;` to create your first migration file.</p>
      <code className="mt-unit-1 font-code-sm text-code-sm text-primary">rift new add_users</code>
    </div>
  )
}

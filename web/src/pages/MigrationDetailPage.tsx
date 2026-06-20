import CodeMirror from '@uiw/react-codemirror'
import { sql } from '@codemirror/lang-sql'
import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ErrorState } from '../components/ErrorState'
import { LoadingSkeleton } from '../components/LoadingSkeleton'
import { fetchLint, fetchMigration } from '../lib/api'
import type { LintWarning } from '../lib/api'
import { useAppStore } from '../stores/appStore'

const schemaTables = ['users', 'accounts', 'sessions', 'orders', 'audit_log']

export function MigrationDetailPage() {
  const { version } = useParams<{ version: string }>()
  const token = useAppStore((state) => state.apiToken)
  const [filter, setFilter] = useState('')

  const migrationQuery = useQuery({
    queryKey: ['migration', version, token],
    queryFn: () => fetchMigration(version ?? '', { token }),
    enabled: Boolean(version && token),
  })

  const lintQuery = useQuery({
    queryKey: ['lint', token],
    queryFn: () => fetchLint({ token }),
    enabled: Boolean(token),
  })

  const migration = migrationQuery.data
  const [sqlText, setSQLText] = useState('')
  const editorValue = sqlText || migration?.up_sql || ''
  const matchingLint = useMemo(() => {
    if (!migration || !lintQuery.data) {
      return []
    }
    return lintQuery.data.results.find((result) => result.filename === migration.filename)?.warnings ?? []
  }, [lintQuery.data, migration])

  const filteredTables = schemaTables.filter((table) => table.includes(filter.toLowerCase()))

  if (!token) {
    return <ErrorState message="API token is required. Save a token on the dashboard or settings page first." />
  }

  if (migrationQuery.isLoading || lintQuery.isLoading) {
    return <LoadingSkeleton />
  }

  const firstError = migrationQuery.error ?? lintQuery.error
  if (firstError instanceof Error) {
    return <ErrorState message={firstError.message} onRetry={() => void Promise.all([migrationQuery.refetch(), lintQuery.refetch()])} />
  }

  if (!migration) {
    return <ErrorState message="Migration was not found." />
  }

  return (
    <div className="grid min-h-[calc(100vh-7rem)] grid-cols-[240px_minmax(0,1fr)_280px] overflow-hidden rounded border border-outline-variant bg-surface-container">
      <aside className="border-r border-outline-variant bg-surface-container p-unit-4">
        <h2 className="font-headline-sm text-headline-sm text-on-surface">Available Schemas</h2>
        <label className="relative mt-unit-4 block">
          <span className="material-symbols-outlined absolute left-2 top-1/2 -translate-y-1/2 text-[18px] text-on-surface-variant">
            search
          </span>
          <input
            className="w-full rounded border border-outline-variant bg-surface-container py-1 pl-8 font-code-sm text-code-sm text-on-surface focus:border-primary focus:ring-primary"
            placeholder="Filter tables"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
          />
        </label>
        <div className="mt-unit-4">
          <div className="mb-unit-2 flex items-center gap-unit-2 font-code-sm text-code-sm text-primary">
            <span className="material-symbols-outlined text-[16px]">folder</span>
            public
          </div>
          <div className="space-y-unit-1">
            {filteredTables.map((table) => (
              <button
                key={table}
                className="flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1 text-left font-code-sm text-code-sm text-on-surface-variant hover:bg-surface-variant/50 hover:text-on-surface"
                type="button"
                onClick={() => setSQLText(`${editorValue}${editorValue.endsWith(' ') || editorValue.endsWith('\n') ? '' : ' '}${table}`)}
              >
                <span className="material-symbols-outlined text-[14px] text-primary">table</span>
                {table}
              </button>
            ))}
          </div>
        </div>
      </aside>

      <section className="min-w-0 bg-surface-container-lowest">
        <div className="flex items-center gap-unit-4 border-b border-outline-variant bg-surface-container-high px-unit-4 py-2">
          <input
            className="min-w-0 flex-1 border-none bg-transparent p-0 font-code-md text-code-md text-on-surface focus:ring-0"
            readOnly
            value={migration.filename}
          />
          <select className="rounded border border-outline-variant bg-surface-container-high font-body-md text-sm text-on-surface focus:border-primary focus:ring-primary" defaultValue="migration">
            <option value="migration">Migration</option>
            <option value="hotfix">Hotfix</option>
            <option value="data">Data</option>
          </select>
          <span className="inline-flex items-center gap-unit-2 font-code-sm text-code-sm text-on-surface-variant">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary-container text-on-primary-container">
              {(migration.applied_by ?? 'L').slice(0, 1).toUpperCase()}
            </span>
            {migration.applied_by ?? 'local'}
          </span>
        </div>
        <CodeMirror
          basicSetup={{ lineNumbers: true, foldGutter: true }}
          className="min-h-[calc(100vh-10rem)] font-code-md text-code-md"
          extensions={[sql()]}
          height="calc(100vh - 10rem)"
          theme="dark"
          value={editorValue}
          onChange={setSQLText}
        />
      </section>

      <LinterPanel warnings={matchingLint} />
    </div>
  )
}

function LinterPanel({ warnings }: { warnings: LintWarning[] }) {
  return (
    <aside className="border-l border-outline-variant bg-surface-container p-unit-4">
      <div className="flex items-center justify-between">
        <h2 className="font-headline-sm text-headline-sm text-on-surface">Zero-Downtime Linter</h2>
        <span className={warnings.length > 0 ? 'font-code-sm text-code-sm text-error' : 'font-code-sm text-code-sm text-secondary'}>
          {warnings.length} Issue{warnings.length === 1 ? '' : 's'}
        </span>
      </div>

      {warnings.length > 0 ? (
        <div className="mt-unit-4 space-y-unit-3">
          {warnings.map((warning) => (
            <div key={`${warning.pattern}-${warning.line}`} className="relative overflow-hidden rounded border border-error/50 bg-surface-container-high p-3">
              <div className="absolute left-0 top-0 h-full w-1 bg-error" />
              <div className="flex gap-unit-2">
                <span className="material-symbols-outlined mt-0.5 text-error">error</span>
                <div>
                  <p className="font-body-md text-body-md font-semibold text-error">{warning.pattern}</p>
                  <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">{warning.message}</p>
                  <p className="mt-unit-2 font-code-sm text-code-sm text-tertiary">Line {warning.line}</p>
                  <button className="mt-unit-3 rounded border border-error/30 bg-error/10 px-3 py-1.5 font-label-caps text-label-caps uppercase text-error hover:bg-error/20" type="button">
                    Auto-Fix
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="mt-unit-4 space-y-unit-3 border-t border-outline-variant pt-unit-4">
          {['Transactional DDL', 'Concurrent indexes', 'No destructive drops'].map((rule) => (
            <div key={rule} className="flex items-center gap-2 font-body-md text-body-md text-on-surface-variant">
              <span className="h-1.5 w-1.5 rounded-full bg-secondary" />
              {rule} passed
            </div>
          ))}
        </div>
      )}
    </aside>
  )
}

import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { ErrorState } from '../components/ErrorState'
import { LoadingSkeleton } from '../components/LoadingSkeleton'
import { SQLDiffPane } from '../components/SQLDiffPane'
import type { SQLDiffLine } from '../components/SQLDiffPane'
import { fetchDiff, fetchLint, fetchMigration, triggerUp } from '../lib/api'
import type { ApplyStreamEvent, ColumnDef, LintWarning, SchemaDiff } from '../lib/api'
import { useAppStore } from '../stores/appStore'

export function MigrationDiffPage() {
  const { version } = useParams<{ version: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const token = useAppStore((state) => state.apiToken)
  const queryClient = useQueryClient()
  const [safePreview, setSafePreview] = useState(true)
  const [forceApply, setForceApply] = useState(false)
  const [events, setEvents] = useState<ApplyStreamEvent[]>([])

  const migrationQuery = useQuery({
    queryKey: ['migration', version, token],
    queryFn: () => fetchMigration(version ?? '', { token }),
    enabled: Boolean(version && token),
  })
  const diffQuery = useQuery({
    queryKey: ['migration-diff', version, token],
    queryFn: () => fetchDiff(version ?? '', { token }),
    enabled: Boolean(version && token),
  })
  const lintQuery = useQuery({
    queryKey: ['lint', token],
    queryFn: () => fetchLint({ token }),
    enabled: Boolean(token),
  })

  const migration = migrationQuery.data
  const diff = diffQuery.data
  const warnings = useMemo(() => {
    if (!migration || !lintQuery.data) {
      return []
    }
    return lintQuery.data.results.find((result) => result.filename === migration.filename)?.warnings ?? []
  }, [lintQuery.data, migration])
  const hasLintErrors = warnings.some((warning) => warning.severity === 'error')

  const applyMutation = useMutation({
    mutationFn: async () => {
      setEvents([])
      return triggerUp(token, forceApply, (event) => setEvents((current) => [...current, event]))
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['status'] }),
        queryClient.invalidateQueries({ queryKey: ['migrations'] }),
        queryClient.invalidateQueries({ queryKey: ['migration-diff'] }),
        queryClient.invalidateQueries({ queryKey: ['history'] }),
      ])
    },
  })

  if (!token) {
    return <ErrorState message="API token is required. Save a token on the dashboard or settings page first." />
  }

  if (migrationQuery.isLoading || diffQuery.isLoading || lintQuery.isLoading) {
    return <LoadingSkeleton />
  }

  const firstError = migrationQuery.error ?? diffQuery.error ?? lintQuery.error
  if (firstError instanceof Error) {
    return <ErrorState message={firstError.message} onRetry={() => void Promise.all([migrationQuery.refetch(), diffQuery.refetch(), lintQuery.refetch()])} />
  }

  if (!migration || !diff) {
    return <ErrorState message="Migration diff was not found." />
  }

  const summary = summarizeDiff(diff)
  const localLines = buildLocalLines(migration.up_sql ?? '', diff)
  const liveLines = buildLiveLines(diff)
  const modalOpen = searchParams.get('apply') === '1'

  function openApplyModal() {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set('apply', '1')
    setSearchParams(nextParams, { replace: false })
  }

  function closeApplyModal() {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.delete('apply')
    setSearchParams(nextParams, { replace: true })
  }

  return (
    <div className="-m-unit-8 flex h-[calc(100vh-3.5rem)] flex-col overflow-hidden bg-background">
      <section className="flex items-center justify-between border-b border-outline-variant bg-surface-container p-unit-4">
        <div className="flex items-center gap-unit-4">
          <span className="material-symbols-outlined text-primary">difference</span>
          <div>
            <h2 className="font-headline-sm text-headline-sm text-on-surface">Schema Summary</h2>
            <p className="mt-1 font-code-sm text-code-sm text-on-surface-variant">
              <span className="text-secondary">{summary.additions} additions</span>
              <span className="mx-2">•</span>
              <span className="text-error">{summary.deletions} deletions</span>
              <span className="mx-2">•</span>
              <span className="text-tertiary">{summary.modifications} modifications</span>
            </p>
          </div>
        </div>
        <div className="flex items-center gap-unit-4">
          <label className="flex cursor-pointer items-center gap-unit-2 font-label-caps text-label-caps uppercase text-on-surface-variant">
            <input
              checked={safePreview}
              className="peer sr-only"
              type="checkbox"
              onChange={(event) => setSafePreview(event.target.checked)}
            />
            <span className="relative h-5 w-10 rounded-full border border-outline-variant bg-surface-variant transition-colors peer-checked:border-primary/50 peer-checked:bg-primary/30">
              <span className="absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-on-surface-variant transition-transform peer-checked:translate-x-5" />
            </span>
            Safe Preview (DDL)
          </label>
          <button
            className="rounded bg-primary px-3 py-1.5 font-label-caps text-label-caps font-bold uppercase text-on-primary transition-colors hover:bg-primary/90 active:scale-95"
            type="button"
            onClick={openApplyModal}
          >
            Apply Migrations
          </button>
        </div>
      </section>

      <div className="flex min-h-0 flex-1 divide-x divide-outline-variant">
        <SQLDiffPane dot="local" lines={localLines} subtitle={migration.filename} title="LOCAL MIGRATIONS" />
        <SQLDiffPane dot="live" lines={liveLines} subtitle="schema: public" title="LIVE DATABASE" />
      </div>

      {modalOpen ? (
        <ApplyConfirmationModal
          applyDisabled={hasLintErrors && !forceApply}
          applyError={applyMutation.error instanceof Error ? applyMutation.error.message : undefined}
          applying={applyMutation.isPending}
          diffSummary={summary}
          events={events}
          filename={migration.filename}
          forceApply={forceApply}
          warnings={warnings}
          onApply={() => applyMutation.mutate()}
          onClose={closeApplyModal}
          onForceApplyChange={setForceApply}
        />
      ) : null}
    </div>
  )
}

type DiffSummary = {
  additions: number
  deletions: number
  modifications: number
}

function summarizeDiff(diff: SchemaDiff): DiffSummary {
  const columnAdditions = diff.TablesModified.reduce((total, table) => total + table.ColumnsAdded.length, 0)
  const columnDeletions = diff.TablesModified.reduce((total, table) => total + table.ColumnsDropped.length, 0)
  const columnModifications = diff.TablesModified.reduce((total, table) => total + table.ColumnsModified.length, 0)
  const indexAdditions = diff.IndexChanges.filter((index) => index.Kind === 'added').length
  const indexDeletions = diff.IndexChanges.filter((index) => index.Kind === 'dropped').length
  const indexModifications = diff.IndexChanges.filter((index) => index.Kind === 'modified').length

  return {
    additions: diff.TablesAdded.length + columnAdditions + indexAdditions,
    deletions: diff.TablesDropped.length + columnDeletions + indexDeletions,
    modifications: diff.TablesModified.length + columnModifications + indexModifications,
  }
}

function buildLocalLines(sql: string, diff: SchemaDiff): SQLDiffLine[] {
  const rawLines = sql.split('\n').filter((line) => line.trim().length > 0)
  const lines = rawLines.map((content, index) => ({ number: index + 1, content, type: inferLocalLineType(content) }))
  if (lines.length > 0) {
    return lines
  }
  return diff.TablesAdded.flatMap((table, index) => tableToLines(table, index + 1, 'addition'))
}

function buildLiveLines(diff: SchemaDiff): SQLDiffLine[] {
  const lines: SQLDiffLine[] = []
  let number = 1
  for (const table of diff.TablesDropped) {
    lines.push(...tableToLines(table, number, 'deletion'))
    number = lines.length + 1
  }
  for (const table of diff.TablesAdded) {
    lines.push({ number: number++, content: `/* table ${table.Name} does not exist in live database */`, type: 'comment' })
  }
  for (const table of diff.TablesModified) {
    for (const column of table.ColumnsAdded) {
      lines.push({ number: number++, content: `/* column ${table.TableName}.${column.Name} missing from live database */`, type: 'comment' })
    }
    for (const column of table.ColumnsDropped) {
      lines.push({ number: number++, content: `ALTER TABLE ${table.TableName} ADD COLUMN ${column.Name} ${describeColumn(column)};`, type: 'deletion' })
    }
    for (const column of table.ColumnsModified) {
      lines.push({ number: number++, content: `ALTER TABLE ${table.TableName} ALTER COLUMN ${column.Before.Name} TYPE ${describeColumn(column.Before)};`, type: 'deletion' })
      lines.push({ number: number++, content: `/* becomes ${describeColumn(column.After)} after migration */`, type: 'comment' })
    }
  }
  for (const index of diff.IndexChanges) {
    if (index.Kind === 'added') {
      lines.push({ number: number++, content: `/* index ${index.Name} missing from live database */`, type: 'comment' })
    } else if (index.Before) {
      lines.push({ number: number++, content: index.Before.Definition, type: index.Kind === 'dropped' ? 'deletion' : 'unchanged' })
    }
  }
  return lines
}

function tableToLines(table: { Name: string; Columns: ColumnDef[] }, start: number, type: SQLDiffLine['type']): SQLDiffLine[] {
  const lines: SQLDiffLine[] = [{ number: start, content: `CREATE TABLE ${table.Name} (`, type }]
  table.Columns.forEach((column, index) => {
    const suffix = index === table.Columns.length - 1 ? '' : ','
    lines.push({ number: start + index + 1, content: `  ${column.Name} ${describeColumn(column)}${suffix}`, type })
  })
  lines.push({ number: start + table.Columns.length + 1, content: ');', type })
  return lines
}

function inferLocalLineType(content: string): SQLDiffLine['type'] {
  const normalized = content.trim().toUpperCase()
  if (normalized.startsWith('DROP')) {
    return 'deletion'
  }
  if (normalized.startsWith('ALTER') || normalized.startsWith('CREATE')) {
    return 'addition'
  }
  return 'unchanged'
}

function describeColumn(column: ColumnDef): string {
  const type = column.MaxLength ? `${column.DataType}(${column.MaxLength})` : column.DataType
  const nullable = column.Nullable ? 'NULL' : 'NOT NULL'
  const defaultValue = column.Default ? ` DEFAULT ${column.Default}` : ''
  return `${type} ${nullable}${defaultValue}`
}

type ApplyConfirmationModalProps = {
  filename: string
  diffSummary: DiffSummary
  warnings: LintWarning[]
  events: ApplyStreamEvent[]
  applying: boolean
  applyDisabled: boolean
  applyError?: string
  forceApply: boolean
  onForceApplyChange: (force: boolean) => void
  onApply: () => void
  onClose: () => void
}

function ApplyConfirmationModal({
  filename,
  diffSummary,
  warnings,
  events,
  applying,
  applyDisabled,
  applyError,
  forceApply,
  onForceApplyChange,
  onApply,
  onClose,
}: ApplyConfirmationModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/70 p-unit-8 backdrop-blur-sm">
      <section className="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-xl border border-outline-variant bg-surface-container-highest p-unit-6">
        <div className="flex items-start justify-between gap-unit-4">
          <div>
            <h2 className="font-headline-md text-headline-md text-on-surface">Apply migration</h2>
            <p className="mt-unit-1 font-code-sm text-code-sm text-primary">{filename}</p>
          </div>
          <button className="text-on-surface-variant hover:text-on-surface" type="button" onClick={onClose}>
            <span className="material-symbols-outlined">close</span>
          </button>
        </div>

        <div className="mt-unit-4 grid gap-unit-3 md:grid-cols-3">
          <SummaryChip label="Additions" tone="success" value={diffSummary.additions} />
          <SummaryChip label="Deletions" tone="error" value={diffSummary.deletions} />
          <SummaryChip label="Modifications" tone="warning" value={diffSummary.modifications} />
        </div>

        <div className="mt-unit-4 rounded border border-outline-variant bg-surface-container p-unit-4">
          <div className="flex items-center justify-between">
            <h3 className="font-headline-sm text-headline-sm text-on-surface">Linter warnings</h3>
            <span className={warnings.length > 0 ? 'font-code-sm text-code-sm text-error' : 'font-code-sm text-code-sm text-secondary'}>
              {warnings.length} issue{warnings.length === 1 ? '' : 's'}
            </span>
          </div>
          {warnings.length > 0 ? (
            <div className="mt-unit-3 space-y-unit-2">
              {warnings.map((warning) => (
                <div key={`${warning.pattern}-${warning.line}`} className="rounded border border-error/50 bg-error-container/10 p-unit-3">
                  <p className="font-body-md text-body-md font-semibold text-error">{warning.pattern} line {warning.line}</p>
                  <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">{warning.message}</p>
                  <p className="mt-unit-2 font-code-sm text-code-sm text-tertiary">{warning.suggestion}</p>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-unit-3 font-body-md text-body-md text-on-surface-variant">No dangerous DDL patterns detected.</p>
          )}
          {warnings.some((warning) => warning.severity === 'error') ? (
            <label className="mt-unit-4 flex items-center gap-unit-2 font-body-md text-body-md text-on-surface-variant">
              <input
                checked={forceApply}
                className="rounded border-outline-variant bg-surface-container-low text-primary focus:ring-primary"
                type="checkbox"
                onChange={(event) => onForceApplyChange(event.target.checked)}
              />
              Force apply after manual review
            </label>
          ) : null}
        </div>

        <DeployLogPanel applyError={applyError} events={events} />

        <div className="mt-unit-6 flex justify-end gap-unit-2">
          <button className="rounded border border-outline-variant px-4 py-2 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant" type="button" onClick={onClose}>
            Cancel
          </button>
          <button
            className="rounded bg-primary px-4 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary disabled:cursor-not-allowed disabled:opacity-40"
            disabled={applyDisabled || applying}
            type="button"
            onClick={onApply}
          >
            {applying ? 'Applying…' : 'Apply'}
          </button>
        </div>
      </section>
    </div>
  )
}

function SummaryChip({ label, value, tone }: { label: string; value: number; tone: 'success' | 'error' | 'warning' }) {
  const toneClass = {
    success: 'border-secondary/30 text-secondary bg-secondary-container/10',
    error: 'border-error/50 text-error bg-error-container/10',
    warning: 'border-tertiary/30 text-tertiary bg-tertiary-container/10',
  }[tone]
  return (
    <div className={`rounded border p-unit-3 ${toneClass}`}>
      <p className="font-label-caps text-label-caps uppercase">{label}</p>
      <p className="mt-unit-1 font-headline-md text-headline-md">{value}</p>
    </div>
  )
}

function DeployLogPanel({ events, applyError }: { events: ApplyStreamEvent[]; applyError?: string }) {
  return (
    <div className="mt-unit-4 rounded border border-outline-variant bg-surface-container-lowest p-unit-4">
      <h3 className="font-headline-sm text-headline-sm text-on-surface">Deploy log</h3>
      <div className="mt-unit-3 max-h-52 space-y-unit-1 overflow-auto font-code-sm text-code-sm">
        {events.length === 0 && !applyError ? <p className="text-on-surface-variant">Awaiting apply command…</p> : null}
        {events.map((event, index) => (
          <LogLine key={`${event.type}-${index}`} event={event} />
        ))}
        {applyError ? <p className="text-error">{new Date().toLocaleTimeString()} error {applyError}</p> : null}
      </div>
    </div>
  )
}

function LogLine({ event }: { event: ApplyStreamEvent }) {
  const timestamp = new Date().toLocaleTimeString()
  if (event.type === 'migration') {
    return (
      <p className="text-secondary">
        {timestamp} applied {event.data.filename} in {event.data.execution_ms}ms
      </p>
    )
  }
  if (event.type === 'done') {
    return <p className="text-primary">{timestamp} done {event.data.applied} migration(s) applied</p>
  }
  return <p className="text-error">{timestamp} error {event.data.message}</p>
}

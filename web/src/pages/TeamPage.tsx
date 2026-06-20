import { useMemo, useState } from 'react'
import { useQueries } from '@tanstack/react-query'
import { ErrorState } from '../components/ErrorState'
import { LoadingSkeleton } from '../components/LoadingSkeleton'
import { fetchConflicts, fetchHistory, fetchTeam } from '../lib/api'
import type { Conflict, MigrationRecord, TeamMember } from '../lib/api'
import { useAppStore } from '../stores/appStore'

export function TeamPage() {
  const token = useAppStore((state) => state.apiToken)
  const [slackEnabled, setSlackEnabled] = useState(false)
  const [discordEnabled, setDiscordEnabled] = useState(false)

  const [conflictsQuery, historyQuery, teamQuery] = useQueries({
    queries: [
      { queryKey: ['conflicts', token], queryFn: () => fetchConflicts({ token }), enabled: Boolean(token) },
      { queryKey: ['history', token], queryFn: () => fetchHistory({ token }), enabled: Boolean(token) },
      { queryKey: ['team', token], queryFn: () => fetchTeam({ token }), enabled: Boolean(token) },
    ],
  })

  const conflicts = useMemo(() => (conflictsQuery.data as Conflict[] | undefined) ?? [], [conflictsQuery.data])
  const history = useMemo(() => (historyQuery.data as MigrationRecord[] | undefined) ?? [], [historyQuery.data])
  const team = useMemo(() => (teamQuery.data as TeamMember[] | undefined) ?? [], [teamQuery.data])
  const firstError = conflictsQuery.error ?? historyQuery.error ?? teamQuery.error

  const recentHistory = useMemo(() => history.slice(0, 8), [history])

  if (!token) {
    return <ErrorState message="API token is required. Save a token on the dashboard or settings page first." />
  }

  if (conflictsQuery.isLoading || historyQuery.isLoading || teamQuery.isLoading) {
    return <LoadingSkeleton />
  }

  if (firstError instanceof Error) {
    return <ErrorState message={firstError.message} onRetry={() => void Promise.all([conflictsQuery.refetch(), historyQuery.refetch(), teamQuery.refetch()])} />
  }

  return (
    <div>
      <div className="mb-unit-8">
        <h2 className="font-display text-display text-on-background">Team & Deployment History</h2>
        <p className="font-body-md text-body-md text-on-surface-variant">
          Track branch conflicts, recent deploys, and database access responsibilities.
        </p>
      </div>

      <div className="grid gap-unit-4 xl:grid-cols-12">
        <main className="space-y-unit-4 xl:col-span-8">
          <ConflictDetectionCard conflicts={conflicts} />
          <DeploymentHistoryTimeline history={recentHistory} />
        </main>
        <aside className="space-y-unit-4 xl:col-span-4">
          <TeamAccessPanel members={team} />
          <NotificationsPanel
            discordEnabled={discordEnabled}
            slackEnabled={slackEnabled}
            onDiscordEnabledChange={setDiscordEnabled}
            onSlackEnabledChange={setSlackEnabled}
          />
        </aside>
      </div>
    </div>
  )
}

function ConflictDetectionCard({ conflicts }: { conflicts: Conflict[] }) {
  const primaryConflict = conflicts[0]
  const hasConflicts = conflicts.length > 0
  return (
    <section className={`overflow-hidden rounded border bg-surface-container ${hasConflicts ? 'border-error' : 'border-outline-variant'}`}>
      <div className="flex items-center justify-between p-unit-4">
        <div className="flex items-center gap-unit-2">
          <span className={`material-symbols-outlined ${hasConflicts ? 'text-error' : 'text-secondary'}`}>{hasConflicts ? 'warning' : 'check_circle'}</span>
          <h3 className={`font-headline-sm text-headline-sm ${hasConflicts ? 'text-error' : 'text-on-surface'}`}>Conflict Detection</h3>
        </div>
        <span className={`rounded border px-2 py-0.5 font-label-caps text-label-caps uppercase ${hasConflicts ? 'border-error/30 bg-error/20 text-error' : 'border-secondary/30 bg-secondary-container/10 text-secondary'}`}>
          {hasConflicts ? `${conflicts.length} conflict${conflicts.length === 1 ? '' : 's'} detected` : 'No conflicts'}
        </span>
      </div>

      {primaryConflict ? (
        <div className="border-t border-outline-variant p-unit-4">
          <p className="font-body-md text-body-md text-on-surface-variant">{primaryConflict.Message}</p>
          <div className="mt-unit-4 grid gap-unit-3 md:grid-cols-2">
            <BranchCard
              checksum={primaryConflict.DatabaseChecksum}
              filename={primaryConflict.DatabaseFilename || 'recorded in database'}
              label="Database branch"
            />
            <BranchCard
              checksum={primaryConflict.LocalChecksum}
              filename={primaryConflict.LocalFilename || 'local migration file'}
              label="Local branch"
            />
          </div>
          <button className="mt-unit-4 rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface hover:bg-surface-variant" type="button">
            Resolve manually
          </button>
        </div>
      ) : (
        <div className="border-t border-outline-variant p-unit-4 font-body-md text-body-md text-on-surface-variant">
          Local migration files and the database state table are in sync.
        </div>
      )}
    </section>
  )
}

function BranchCard({ label, filename, checksum }: { label: string; filename: string; checksum: string }) {
  return (
    <div className="rounded border border-outline-variant bg-surface-container-low p-unit-3">
      <p className="font-label-caps text-label-caps uppercase text-on-surface-variant">{label}</p>
      <code className="mt-unit-2 block whitespace-pre-wrap break-all font-code-sm text-code-sm text-primary">
        {filename || '—'}
        {'\n'}checksum: {checksum || 'unavailable'}
      </code>
    </div>
  )
}

function DeploymentHistoryTimeline({ history }: { history: MigrationRecord[] }) {
  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <div className="mb-unit-4 flex items-center justify-between">
        <h3 className="font-headline-sm text-headline-sm text-on-surface">Deployment History</h3>
        <button className="font-label-caps text-label-caps uppercase text-primary hover:underline" type="button">View All →</button>
      </div>

      {history.length > 0 ? (
        <div className="relative ml-2.5 space-y-unit-6 border-l-2 border-outline-variant pl-6">
          {history.map((record) => (
            <TimelineItem key={`${record.Version}-${record.AppliedAt}`} record={record} />
          ))}
        </div>
      ) : (
        <div className="flex min-h-56 flex-col items-center justify-center text-center">
          <span className="material-symbols-outlined text-5xl text-on-surface-variant/30">history</span>
          <h4 className="mt-unit-4 font-headline-sm text-headline-sm text-on-surface">No deployments yet</h4>
          <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">Applied migrations will appear here.</p>
        </div>
      )}
    </section>
  )
}

function TimelineItem({ record }: { record: MigrationRecord }) {
  const failed = record.RolledBack
  return (
    <article className="relative">
      <span className={`absolute -left-[31px] top-1.5 h-3 w-3 rounded-full ring-4 ring-surface-container ${failed ? 'bg-error' : 'bg-secondary'}`} />
      <div className="flex flex-wrap items-center gap-unit-2">
        <code className="font-code-sm text-code-sm text-primary">{record.Filename}</code>
        <span className={`rounded border px-2 py-0.5 font-label-caps text-label-caps uppercase ${failed ? 'border-error/50 bg-error-container/10 text-error' : 'border-secondary/30 bg-secondary-container/10 text-secondary'}`}>
          {failed ? 'Failed' : 'Success'}
        </span>
      </div>
      <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">
        {failed ? 'Rollback triggered automatically' : 'Applied to Production via CI/CD'}
      </p>
      {failed ? (
        <p className="mt-unit-2 rounded border border-error/30 bg-surface-container-low p-2 font-code-sm text-code-sm text-error">
          Migration was rolled back after deployment.
        </p>
      ) : null}
      <div className="mt-unit-2 flex items-center gap-unit-2 font-body-md text-body-md text-on-surface-variant">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary-container text-on-primary-container font-label-caps text-label-caps">
          {(record.AppliedBy || 'R').slice(0, 1).toUpperCase()}
        </span>
        Deployed by {record.AppliedBy || 'rift'} · {new Date(record.AppliedAt).toLocaleString()}
      </div>
    </article>
  )
}

function TeamAccessPanel({ members }: { members: TeamMember[] }) {
  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <div className="mb-unit-4 flex items-center justify-between">
        <h3 className="font-headline-sm text-headline-sm text-on-surface">Team Access</h3>
        <button className="flex h-8 w-8 items-center justify-center rounded border border-outline-variant text-primary hover:bg-primary/10" type="button">
          <span className="material-symbols-outlined text-[18px]">add</span>
        </button>
      </div>
      <div className="space-y-unit-3">
        {(members.length > 0 ? members : [{ name: 'Local Operator', email: 'local@rift.dev', role: 'Admin' }]).map((member) => (
          <MemberRow key={member.email} member={member} />
        ))}
      </div>
    </section>
  )
}

function MemberRow({ member }: { member: TeamMember }) {
  const role = member.role.toLowerCase()
  const avatarClass = role.includes('admin')
    ? 'bg-primary-container text-on-primary-container'
    : role.includes('developer')
      ? 'bg-tertiary-container text-on-tertiary-container'
      : 'bg-surface-variant text-on-surface-variant'
  return (
    <div className="flex items-center gap-unit-3">
      <span className={`flex h-9 w-9 items-center justify-center rounded-full font-label-caps text-label-caps uppercase ${avatarClass}`}>
        {initials(member.name)}
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate font-body-md text-body-md text-on-surface">{member.name}</p>
        <p className="truncate font-code-sm text-code-sm text-on-surface-variant">{member.email}</p>
      </div>
      <span className="rounded border border-outline-variant px-2 py-0.5 font-label-caps text-label-caps uppercase text-on-surface-variant">
        {member.role}
      </span>
    </div>
  )
}

function NotificationsPanel({
  slackEnabled,
  discordEnabled,
  onSlackEnabledChange,
  onDiscordEnabledChange,
}: {
  slackEnabled: boolean
  discordEnabled: boolean
  onSlackEnabledChange: (enabled: boolean) => void
  onDiscordEnabledChange: (enabled: boolean) => void
}) {
  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <h3 className="font-headline-sm text-headline-sm text-on-surface">Notifications / Webhooks</h3>
      <div className="mt-unit-4 space-y-unit-4">
        <WebhookToggle enabled={slackEnabled} label="Slack" onEnabledChange={onSlackEnabledChange} />
        <WebhookToggle enabled={discordEnabled} label="Discord" onEnabledChange={onDiscordEnabledChange} />
      </div>
    </section>
  )
}

function WebhookToggle({ label, enabled, onEnabledChange }: { label: string; enabled: boolean; onEnabledChange: (enabled: boolean) => void }) {
  return (
    <div>
      <label className="flex cursor-pointer items-center justify-between gap-unit-4">
        <span className="font-body-md text-body-md text-on-surface">{label}</span>
        <input checked={enabled} className="peer sr-only" type="checkbox" onChange={(event) => onEnabledChange(event.target.checked)} />
        <span className="relative h-5 w-10 rounded-full border border-outline-variant bg-surface-variant transition-colors peer-checked:border-primary/50 peer-checked:bg-primary/30">
          <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-on-surface-variant transition-transform ${enabled ? 'left-5' : 'left-0.5'}`} />
        </span>
      </label>
      {enabled ? (
        <input
          className="mt-unit-2 w-full rounded border border-outline-variant bg-surface-container-low px-3 py-2 font-code-sm text-code-sm text-on-surface placeholder:text-on-surface-variant focus:border-primary focus:ring-primary"
          placeholder={`${label} webhook URL`}
        />
      ) : null}
    </div>
  )
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) {
    return '?'
  }
  return parts.slice(0, 2).map((part) => part[0]?.toUpperCase() ?? '').join('')
}

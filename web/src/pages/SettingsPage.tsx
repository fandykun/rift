import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ErrorState } from '../components/ErrorState'
import { fetchStatus } from '../lib/api'
import { useAppStore } from '../stores/appStore'

export function SettingsPage() {
  const apiToken = useAppStore((state) => state.apiToken)
  const environmentName = useAppStore((state) => state.environmentName)
  const setApiToken = useAppStore((state) => state.setApiToken)
  const setEnvironmentName = useAppStore((state) => state.setEnvironmentName)
  const [draftToken, setDraftToken] = useState(apiToken)
  const [draftEnvironment, setDraftEnvironment] = useState(environmentName)
  const [tokenVisible, setTokenVisible] = useState(false)
  const [saved, setSaved] = useState(false)

  const statusQuery = useQuery({
    queryKey: ['status', apiToken],
    queryFn: () => fetchStatus({ token: apiToken }),
    enabled: apiToken.length > 0,
    retry: false,
  })

  const connection = useMemo(() => {
    if (!apiToken) {
      return {
        tone: 'warning' as const,
        icon: 'key_off',
        label: 'Token Required',
        description: 'Save the bearer token configured in rift.yaml to check the database connection.',
      }
    }
    if (statusQuery.isFetching) {
      return {
        tone: 'neutral' as const,
        icon: 'sync',
        label: 'Checking Connection',
        description: 'Rift is querying the API status endpoint.',
      }
    }
    if (statusQuery.error instanceof Error) {
      return {
        tone: 'error' as const,
        icon: 'error',
        label: 'Disconnected',
        description: statusQuery.error.message,
      }
    }
    if (statusQuery.data) {
      const { counts } = statusQuery.data
      return {
        tone: 'success' as const,
        icon: 'check_circle',
        label: 'Connected',
        description: `${statusQuery.data.environment} · ${counts.applied} applied · ${counts.pending} pending · ${counts.rolled_back} rolled back`,
      }
    }
    return {
      tone: 'neutral' as const,
      icon: 'radio_button_unchecked',
      label: 'Not Checked',
      description: 'Connection status has not been checked yet.',
    }
  }, [apiToken, statusQuery.data, statusQuery.error, statusQuery.isFetching])

  const saveDisabled = draftToken.trim() === apiToken && draftEnvironment.trim() === environmentName

  function saveSettings() {
    setApiToken(draftToken.trim())
    setEnvironmentName(draftEnvironment.trim() || 'development')
    setSaved(true)
    window.setTimeout(() => setSaved(false), 2_000)
  }

  return (
    <div>
      <div className="mb-unit-8">
        <h2 className="font-display text-display text-on-background">Settings</h2>
        <p className="font-body-md text-body-md text-on-surface-variant">
          Configure local dashboard access and verify the active Rift API connection.
        </p>
      </div>

      <div className="grid gap-unit-4 xl:grid-cols-12">
        <section className="rounded border border-outline-variant bg-surface-container p-unit-6 xl:col-span-8">
          <div className="flex items-start justify-between gap-unit-4 border-b border-outline-variant pb-unit-4">
            <div>
              <h3 className="font-headline-sm text-headline-sm text-on-surface">API Access</h3>
              <p className="mt-unit-1 font-body-md text-body-md text-on-surface-variant">
                These settings are stored locally in your browser via the Rift dashboard store.
              </p>
            </div>
            {saved ? (
              <span className="rounded border border-secondary/30 bg-secondary-container/10 px-2 py-1 font-label-caps text-label-caps uppercase text-secondary">
                Saved
              </span>
            ) : null}
          </div>

          <div className="mt-unit-6 space-y-unit-6">
            <label className="block">
              <span className="font-label-caps text-label-caps uppercase text-on-surface-variant">API Token</span>
              <div className="mt-unit-2 flex gap-unit-2">
                <input
                  className="min-w-0 flex-1 rounded border border-outline-variant bg-surface-container-low px-3 py-2 font-code-sm text-code-sm text-on-surface placeholder:text-on-surface-variant focus:border-primary focus:ring-primary"
                  placeholder="Bearer token from rift.yaml"
                  type={tokenVisible ? 'text' : 'password'}
                  value={draftToken}
                  onChange={(event) => setDraftToken(event.target.value)}
                />
                <button
                  className="rounded border border-outline-variant px-3 py-2 text-on-surface-variant transition-colors hover:bg-surface-variant hover:text-on-surface"
                  type="button"
                  aria-label={tokenVisible ? 'Hide API token' : 'Reveal API token'}
                  onClick={() => setTokenVisible((visible) => !visible)}
                >
                  <span className="material-symbols-outlined text-[20px]">{tokenVisible ? 'visibility_off' : 'visibility'}</span>
                </button>
              </div>
              <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">
                Sent as an Authorization bearer token on all API requests.
              </p>
            </label>

            <label className="block">
              <span className="font-label-caps text-label-caps uppercase text-on-surface-variant">Environment Name</span>
              <input
                className="mt-unit-2 w-full rounded border border-outline-variant bg-surface-container-low px-3 py-2 font-body-md text-body-md text-on-surface placeholder:text-on-surface-variant focus:border-primary focus:ring-primary"
                placeholder="development"
                value={draftEnvironment}
                onChange={(event) => setDraftEnvironment(event.target.value)}
              />
              <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">
                Used in the sidebar label. The API-reported environment remains visible in the status card.
              </p>
            </label>
          </div>

          <div className="mt-unit-8 flex flex-wrap items-center justify-between gap-unit-3 border-t border-outline-variant pt-unit-4">
            <button
              className="rounded bg-primary px-4 py-2 font-label-caps text-label-caps font-bold uppercase text-on-primary transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40"
              disabled={saveDisabled}
              type="button"
              onClick={saveSettings}
            >
              Save Settings
            </button>
            <p className="font-code-sm text-code-sm text-on-surface-variant">localStorage key: rift-app-state</p>
          </div>
        </section>

        <aside className="space-y-unit-4 xl:col-span-4">
          <ConnectionStatusCard
            description={connection.description}
            icon={connection.icon}
            label={connection.label}
            tone={connection.tone}
            onRetry={() => void statusQuery.refetch()}
          />
          {statusQuery.error instanceof Error ? <ErrorState message={statusQuery.error.message} onRetry={() => void statusQuery.refetch()} /> : null}
        </aside>
      </div>
    </div>
  )
}

type ConnectionTone = 'success' | 'warning' | 'error' | 'neutral'

type ConnectionStatusCardProps = {
  icon: string
  label: string
  description: string
  tone: ConnectionTone
  onRetry: () => void
}

function ConnectionStatusCard({ icon, label, description, tone, onRetry }: ConnectionStatusCardProps) {
  const toneClass = {
    success: 'border-secondary/30 bg-secondary-container/10 text-secondary',
    warning: 'border-tertiary/30 bg-tertiary-container/10 text-tertiary',
    error: 'border-error/50 bg-error-container/10 text-error',
    neutral: 'border-outline-variant bg-surface-container-low text-on-surface-variant',
  }[tone]

  return (
    <section className="rounded border border-outline-variant bg-surface-container p-unit-4">
      <div className={`inline-flex items-center gap-unit-2 rounded border px-2 py-1 font-label-caps text-label-caps uppercase ${toneClass}`}>
        <span className="material-symbols-outlined text-[18px]">{icon}</span>
        {label}
      </div>
      <h3 className="mt-unit-4 font-headline-sm text-headline-sm text-on-surface">Database Connection</h3>
      <p className="mt-unit-2 font-body-md text-body-md text-on-surface-variant">{description}</p>
      <button
        className="mt-unit-4 rounded border border-outline-variant px-3 py-1.5 font-label-caps text-label-caps uppercase text-on-surface transition-colors hover:bg-surface-variant"
        type="button"
        onClick={onRetry}
      >
        Check Again
      </button>
    </section>
  )
}

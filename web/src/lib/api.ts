export type MigrationStatus = 'applied' | 'pending' | 'rolled-back' | 'failed'

export type ApiCounts = {
  applied: number
  pending: number
  rolled_back: number
  total: number
}

export type StatusResponse = {
  environment: string
  counts: ApiCounts
  last_deploy?: string
}

export type Migration = {
  version: string
  filename: string
  status: MigrationStatus
  applied_at?: string
  applied_by?: string
  execution_ms?: number
  has_lint: boolean
  up_sql?: string
  down_sql?: string
}

export type LintWarning = {
  pattern: string
  line: number
  severity: 'error' | 'warning'
  message: string
  suggestion: string
}

export type LintResult = {
  filename: string
  warnings: LintWarning[]
}

export type LintResponse = {
  error_count: number
  warning_count: number
  results: LintResult[]
}

export type TeamMember = {
  name: string
  email: string
  role: string
}

export type Conflict = {
  Type: string
  Version: string
  DatabaseFilename: string
  LocalFilename: string
  DatabaseChecksum: string
  LocalChecksum: string
  Message: string
}

export type SchemaDiff = {
  tables_added?: unknown[]
  tables_dropped?: unknown[]
  columns_added?: unknown[]
  columns_dropped?: unknown[]
  columns_modified?: unknown[]
  indexes_added?: unknown[]
  indexes_dropped?: unknown[]
  indexes_modified?: unknown[]
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

type RequestOptions = {
  token?: string
  signal?: AbortSignal
}

async function requestJSON<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: buildHeaders(options.token),
    signal: options.signal,
  })

  if (!response.ok) {
    const message = await readErrorMessage(response)
    throw new Error(message)
  }

  return (await response.json()) as T
}

function buildHeaders(token?: string): HeadersInit {
  if (!token) {
    return {}
  }
  return { Authorization: `Bearer ${token}` }
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as { error?: string }
    return payload.error ?? `Request failed with status ${response.status}`
  } catch {
    return `Request failed with status ${response.status}`
  }
}

export function fetchStatus(options?: RequestOptions): Promise<StatusResponse> {
  return requestJSON<StatusResponse>('/api/v1/status', options)
}

export function fetchMigrations(options?: RequestOptions): Promise<Migration[]> {
  return requestJSON<Migration[]>('/api/v1/migrations', options)
}

export function fetchMigration(version: string, options?: RequestOptions): Promise<Migration> {
  return requestJSON<Migration>(`/api/v1/migrations/${encodeURIComponent(version)}`, options)
}

export function fetchDiff(version: string, options?: RequestOptions): Promise<SchemaDiff> {
  return requestJSON<SchemaDiff>(`/api/v1/migrations/${encodeURIComponent(version)}/diff`, options)
}

export function fetchHistory(options?: RequestOptions): Promise<Migration[]> {
  return requestJSON<Migration[]>('/api/v1/history', options)
}

export function fetchLint(options?: RequestOptions): Promise<LintResponse> {
  return requestJSON<LintResponse>('/api/v1/lint', options)
}

export function fetchTeam(options?: RequestOptions): Promise<TeamMember[]> {
  return requestJSON<TeamMember[]>('/api/v1/team', options)
}

export function fetchConflicts(options?: RequestOptions): Promise<Conflict[]> {
  return requestJSON<Conflict[]>('/api/v1/conflicts', options)
}

export function triggerUp(token?: string): EventSource {
  const query = token ? `?token=${encodeURIComponent(token)}` : ''
  return new EventSource(`${API_BASE_URL}/api/v1/migrate/up${query}`)
}

export async function triggerDown(steps: number, token?: string): Promise<{ status: string; steps: number }> {
  const response = await fetch(`${API_BASE_URL}/api/v1/migrate/down`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...buildHeaders(token),
    },
    body: JSON.stringify({ steps }),
  })

  if (!response.ok) {
    const message = await readErrorMessage(response)
    throw new Error(message)
  }

  return (await response.json()) as { status: string; steps: number }
}

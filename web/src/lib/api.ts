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

export type CreateMigrationRequest = {
  name: string
  up_sql?: string
  down_sql?: string
}

export type MigrationRecord = {
  ID: number
  Version: string
  Filename: string
  Checksum: string
  AppliedAt: string
  AppliedBy: string
  ExecutionMs: number
  RolledBack: boolean
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

type RawLintWarning = Partial<LintWarning> & {
  Pattern?: string
  Line?: number
  Severity?: string
  Message?: string
  Suggestion?: string
}

type RawLintResult = {
  filename: string
  warnings?: RawLintWarning[] | null
}

type RawLintResponse = {
  error_count: number
  warning_count: number
  results?: RawLintResult[] | null
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

export type ColumnDef = {
  TableName: string
  Name: string
  DataType: string
  Nullable: boolean
  Default: string
  MaxLength?: number
}

export type TableDef = {
  Name: string
  Columns: ColumnDef[]
}

export type ColumnModification = {
  Before: ColumnDef
  After: ColumnDef
}

export type TableModification = {
  TableName: string
  ColumnsAdded: ColumnDef[]
  ColumnsDropped: ColumnDef[]
  ColumnsModified: ColumnModification[]
}

export type IndexDef = {
  Name: string
  TableName: string
  Definition: string
}

export type IndexChange = {
  Name: string
  Kind: 'added' | 'dropped' | 'modified'
  Before?: IndexDef
  After?: IndexDef
}

export type SchemaDiff = {
  TablesAdded: TableDef[]
  TablesDropped: TableDef[]
  TablesModified: TableModification[]
  IndexChanges: IndexChange[]
}

export type ApplyMigrationEvent = {
  status: 'applied'
  version: string
  filename: string
  execution_ms: number
}

export type ApplyDoneEvent = {
  status: 'done'
  applied: number
}

export type ApplyErrorEvent = {
  status: 'error'
  message: string
}

export type ApplyStreamEvent =
  | { type: 'migration'; data: ApplyMigrationEvent }
  | { type: 'done'; data: ApplyDoneEvent }
  | { type: 'error'; data: ApplyErrorEvent }

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

async function postJSON<T>(path: string, body: unknown, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...buildHeaders(options.token),
    },
    signal: options.signal,
    body: JSON.stringify(body),
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

export function createMigration(request: CreateMigrationRequest, options?: RequestOptions): Promise<Migration> {
  return postJSON<Migration>('/api/v1/migrations', request, options)
}

export function fetchMigration(version: string, options?: RequestOptions): Promise<Migration> {
  return requestJSON<Migration>(`/api/v1/migrations/${encodeURIComponent(version)}`, options)
}

export async function fetchDiff(version: string, options?: RequestOptions): Promise<SchemaDiff> {
  const diff = await requestJSON<SchemaDiff>(`/api/v1/migrations/${encodeURIComponent(version)}/diff`, options)
  return normalizeSchemaDiff(diff)
}

function normalizeSchemaDiff(diff: SchemaDiff): SchemaDiff {
  return {
    TablesAdded: (diff.TablesAdded ?? []).map(normalizeTableDef),
    TablesDropped: (diff.TablesDropped ?? []).map(normalizeTableDef),
    TablesModified: (diff.TablesModified ?? []).map((table) => ({
      ...table,
      ColumnsAdded: table.ColumnsAdded ?? [],
      ColumnsDropped: table.ColumnsDropped ?? [],
      ColumnsModified: table.ColumnsModified ?? [],
    })),
    IndexChanges: diff.IndexChanges ?? [],
  }
}

function normalizeTableDef(table: TableDef): TableDef {
  return {
    ...table,
    Columns: table.Columns ?? [],
  }
}

export function fetchHistory(options?: RequestOptions): Promise<MigrationRecord[]> {
  return requestJSON<MigrationRecord[]>('/api/v1/history', options)
}

export async function fetchLint(options?: RequestOptions): Promise<LintResponse> {
  const lint = await requestJSON<RawLintResponse>('/api/v1/lint', options)
  return {
    error_count: lint.error_count,
    warning_count: lint.warning_count,
    results: (lint.results ?? []).map((result) => ({
      filename: result.filename,
      warnings: (result.warnings ?? []).map(normalizeLintWarning),
    })),
  }
}

function normalizeLintWarning(warning: RawLintWarning): LintWarning {
  const severity = warning.severity ?? warning.Severity
  return {
    pattern: warning.pattern ?? warning.Pattern ?? 'UNKNOWN_PATTERN',
    line: warning.line ?? warning.Line ?? 0,
    severity: severity === 'error' ? 'error' : 'warning',
    message: warning.message ?? warning.Message ?? 'No message returned by linter.',
    suggestion: warning.suggestion ?? warning.Suggestion ?? 'Review the migration manually before applying.',
  }
}

export function fetchTeam(options?: RequestOptions): Promise<TeamMember[]> {
  return requestJSON<TeamMember[]>('/api/v1/team', options)
}

export function fetchConflicts(options?: RequestOptions): Promise<Conflict[]> {
  return requestJSON<Conflict[]>('/api/v1/conflicts', options)
}

export async function triggerUp(
  token: string | undefined,
  force: boolean,
  onEvent: (event: ApplyStreamEvent) => void,
): Promise<ApplyDoneEvent> {
  const response = await fetch(`${API_BASE_URL}/api/v1/migrate/up${force ? '?force=true' : ''}`, {
    method: 'POST',
    headers: buildHeaders(token),
  })

  if (!response.ok) {
    const message = await readErrorMessage(response)
    throw new Error(message)
  }
  if (!response.body) {
    throw new Error('Migration stream is not available in this browser')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let doneEvent: ApplyDoneEvent | undefined

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    buffer += decoder.decode(value, { stream: true })
    const chunks = buffer.split('\n\n')
    buffer = chunks.pop() ?? ''
    for (const chunk of chunks) {
      const event = parseSSEChunk(chunk)
      if (!event) {
        continue
      }
      onEvent(event)
      if (event.type === 'done') {
        doneEvent = event.data
      }
      if (event.type === 'error') {
        throw new Error(event.data.message)
      }
    }
  }

  if (buffer.trim()) {
    const event = parseSSEChunk(buffer)
    if (event) {
      onEvent(event)
      if (event.type === 'done') {
        doneEvent = event.data
      }
      if (event.type === 'error') {
        throw new Error(event.data.message)
      }
    }
  }

  if (!doneEvent) {
    throw new Error('Migration stream ended without a completion event')
  }
  return doneEvent
}

function parseSSEChunk(chunk: string): ApplyStreamEvent | undefined {
  const eventLine = chunk.split('\n').find((line) => line.startsWith('event:'))
  const dataLine = chunk.split('\n').find((line) => line.startsWith('data:'))
  if (!eventLine || !dataLine) {
    return undefined
  }

  const eventName = eventLine.replace(/^event:\s*/, '').trim()
  const data = JSON.parse(dataLine.replace(/^data:\s*/, '')) as unknown
  if (eventName === 'migration') {
    return { type: 'migration', data: data as ApplyMigrationEvent }
  }
  if (eventName === 'done') {
    return { type: 'done', data: data as ApplyDoneEvent }
  }
  if (eventName === 'error') {
    return { type: 'error', data: data as ApplyErrorEvent }
  }
  return undefined
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

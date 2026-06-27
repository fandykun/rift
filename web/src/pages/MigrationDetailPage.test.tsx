import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchLint, fetchMigration, updateMigration } from '../lib/api'
import { useAppStore } from '../stores/appStore'
import { MigrationDetailPage } from './MigrationDetailPage'

vi.mock('@uiw/react-codemirror', () => ({
  default: ({ value, onChange }: { value: string; onChange: (value: string) => void }) => (
    <textarea aria-label="SQL editor" value={value} onChange={(event) => onChange(event.target.value)} />
  ),
}))

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchMigration: vi.fn(),
    fetchLint: vi.fn(),
    updateMigration: vi.fn(),
  }
})

function renderDetail() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/migrations/20260626_120000"]}>
        <Routes>
          <Route element={<MigrationDetailPage />} path="/migrations/:version" />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('MigrationDetailPage', () => {
  beforeEach(() => {
    localStorage.clear()
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false })
    vi.mocked(fetchMigration).mockResolvedValue({
      version: '20260626_120000',
      filename: '20260626_120000_add_billing_events',
      status: 'pending',
      has_lint: false,
      up_sql: '-- old up\n',
      down_sql: '-- old down\n',
    })
    vi.mocked(fetchLint).mockResolvedValue({ error_count: 0, warning_count: 0, results: [] })
    vi.mocked(updateMigration).mockResolvedValue({
      version: '20260626_120000',
      filename: '20260626_120000_add_billing_events',
      status: 'pending',
      has_lint: false,
      up_sql: 'CREATE TABLE billing_events (id BIGSERIAL PRIMARY KEY);\n',
      down_sql: '-- old down\n',
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('edits and saves the pending up migration file', async () => {
    renderDetail()

    const editor = await screen.findByLabelText('SQL editor')
    expect((editor as HTMLTextAreaElement).value).toBe('-- old up\n')
    expect(screen.getByRole('button', { name: /save files/i }).hasAttribute('disabled')).toBe(true)

    fireEvent.change(editor, { target: { value: 'CREATE TABLE billing_events (id BIGSERIAL PRIMARY KEY);' } })
    fireEvent.click(screen.getByRole('button', { name: /save files/i }))

    await waitFor(() => {
      expect(updateMigration).toHaveBeenCalledWith(
        '20260626_120000',
        {
          up_sql: 'CREATE TABLE billing_events (id BIGSERIAL PRIMARY KEY);',
          down_sql: '-- old down\n',
        },
        { token: 'token' },
      )
    })
    expect(await screen.findByText('Migration files saved')).not.toBeNull()
  })

  it('switches to and saves the down migration file', async () => {
    renderDetail()

    await screen.findByLabelText('SQL editor')
    fireEvent.click(screen.getByRole('button', { name: /down.sql/i }))
    const editor = screen.getByLabelText('SQL editor')
    expect((editor as HTMLTextAreaElement).value).toBe('-- old down\n')

    fireEvent.change(editor, { target: { value: 'DROP TABLE billing_events;' } })
    fireEvent.click(screen.getByRole('button', { name: /save files/i }))

    await waitFor(() => {
      expect(updateMigration).toHaveBeenCalledWith(
        '20260626_120000',
        {
          up_sql: '-- old up\n',
          down_sql: 'DROP TABLE billing_events;',
        },
        { token: 'token' },
      )
    })
  })
})

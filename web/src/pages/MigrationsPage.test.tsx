import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchLint, fetchMigrations, fetchStatus } from '../lib/api'
import { useAppStore } from '../stores/appStore'
import { MigrationsPage } from './MigrationsPage'

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchStatus: vi.fn(),
    fetchMigrations: vi.fn(),
    fetchLint: vi.fn(),
  }
})

function renderWithProviders() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MigrationsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('MigrationsPage', () => {
  beforeEach(() => {
    localStorage.clear()
    useAppStore.setState({ apiToken: 'token', environmentName: 'development', sidebarOpen: false })
    vi.mocked(fetchStatus).mockResolvedValue({
      environment: 'test',
      counts: { total: 2, applied: 1, pending: 1, rolled_back: 0 },
    })
    vi.mocked(fetchMigrations).mockResolvedValue([
      {
        version: '20260621_120000',
        filename: '20260621_120000_create_accounts.up.sql',
        status: 'pending',
        applied_by: 'local',
        has_lint: true,
      },
      {
        version: '20260620_180000',
        filename: '20260620_180000_create_widgets.up.sql',
        status: 'applied',
        applied_at: '2026-06-20T10:00:00Z',
        applied_by: 'ci',
        execution_ms: 42,
        has_lint: false,
      },
    ])
    vi.mocked(fetchLint).mockResolvedValue({
      error_count: 1,
      warning_count: 0,
      results: [
        {
          filename: '20260621_120000_create_accounts.up.sql',
          warnings: [
            {
              pattern: 'DROP_COLUMN',
              line: 3,
              severity: 'error',
              message: 'Dropping a column is irreversible.',
              suggestion: 'Rename the column first.',
            },
          ],
        },
      ],
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders migration rows, counts, linter alerts, and action links from mocked API data', async () => {
    renderWithProviders()

    expect(await screen.findByText('create_accounts')).not.toBeNull()
    expect(screen.getByText('create_widgets')).not.toBeNull()
    expect(screen.getByText('Total Migrations')).not.toBeNull()
    expect(screen.getByText('Applied')).not.toBeNull()
    expect(screen.getByText('Pending')).not.toBeNull()
    expect(screen.getByText('DROP_COLUMN')).not.toBeNull()
    expect(screen.getAllByRole('link', { name: 'View' })[0]?.getAttribute('href')).toBe('/migrations/20260621_120000')
    expect(screen.getAllByRole('link', { name: 'Diff' })[0]?.getAttribute('href')).toBe('/migrations/20260621_120000/diff')

    await waitFor(() => expect(useAppStore.getState().environmentName).toBe('test'))
  })
})
